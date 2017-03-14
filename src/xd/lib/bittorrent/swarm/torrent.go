package swarm

import (
	"bytes"
	"encoding/binary"
	"net"
	"sync"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/metainfo"
	"xd/lib/network"
	"xd/lib/storage"
	"xd/lib/tracker"
)

// single torrent tracked in a swarm
type Torrent struct {
	// network context
	Net       network.Network
	Trackers  []tracker.Announcer
	announcer *time.Ticker
	// our peer id
	id    common.PeerID
	st    storage.Torrent
	piece chan pieceEvent
	// active connections
	conns map[string]*PeerConn
	cmtx  sync.RWMutex
	// piece tracker
	pt *pieceTracker
}

func newTorrent(st storage.Torrent) *Torrent {
	t := &Torrent{
		st:    st,
		piece: make(chan pieceEvent, 8),
		conns: make(map[string]*PeerConn),
		pt:    createPieceTracker(st),
	}
	t.pt.have = t.broadcastHave
	return t
}

func (t *Torrent) GetStatus() *TorrentStatus {
	t.cmtx.Lock()
	var peers []*PeerConnStats
	for _, conn := range t.conns {
		if conn != nil {
			peers = append(peers, conn.Stats())
		}
	}
	t.cmtx.Unlock()
	return &TorrentStatus{
		Peers: peers,
	}
}

func (t *Torrent) Bitfield() *bittorrent.Bitfield {
	return t.st.Bitfield()
}

// start annoucing on all trackers
func (t *Torrent) StartAnnouncing() {
	for _, tr := range t.Trackers {
		t.Announce(tr, "started")
	}
	if t.announcer == nil {
		t.announcer = time.NewTicker(time.Second)
	}
	go t.pollAnnounce()
}

// stop annoucing on all trackers
func (t *Torrent) StopAnnouncing() {
	if t.announcer != nil {
		t.announcer.Stop()
	}
	for _, tr := range t.Trackers {
		t.Announce(tr, "stopped")
	}
}

// poll announce ticker channel and issue announces
func (t *Torrent) pollAnnounce() {
	for {
		_, ok := <-t.announcer.C
		if !ok {
			// done
			return
		}
		for _, tr := range t.Trackers {
			if tr.ShouldAnnounce() {
				go t.Announce(tr, "")
			}
		}
	}
}

// do an announce
func (t *Torrent) Announce(tr tracker.Announcer, event string) {
	req := &tracker.Request{
		Infohash: t.st.Infohash(),
		PeerID:   t.id,
		IP:       t.Net.Addr(),
		Port:     6881,
		Event:    event,
		NumWant:  10, // TODO: don't hardcode
		Left:     t.st.DownloadRemaining(),
		Compact:  true,
	}
	resp, err := tr.Announce(req)
	if err == nil {
		for _, p := range resp.Peers {
			a, e := p.Resolve(t.Net)
			if e == nil {
				if a.String() == t.Net.Addr().String() || t.HasConn(a) {
					// don't connect to self or a duplicate
					continue
				}
				t.cmtx.Lock()
				t.conns[a.String()] = nil
				t.cmtx.Unlock()
				// no error resolving
				go t.PersistPeer(a, p.ID)
			} else {
				log.Warnf("failed to resolve peer %s", e.Error())
			}
		}
	} else {
		log.Warnf("failed to announce to %s", tr.Name())
	}
}

// persit a connection to a peer
func (t *Torrent) PersistPeer(a net.Addr, id common.PeerID) {

	triesLeft := 10
	for !t.Done() {
		err := t.AddPeer(a, id)
		if err != nil {
			triesLeft--
		} else {
			return
		}
		if triesLeft == 0 {
			return
		}
	}
}

func (t *Torrent) HasConn(a net.Addr) (has bool) {
	t.cmtx.Lock()
	defer t.cmtx.Unlock()
	_, has = t.conns[a.String()]
	return
}

// connect to a new peer for this swarm, blocks
func (t *Torrent) AddPeer(a net.Addr, id common.PeerID) error {
	c, err := t.Net.Dial(a.Network(), a.String())
	if err == nil {
		// connected
		ih := t.st.Infohash()
		// build handshake
		h := new(bittorrent.Handshake)
		copy(h.Infohash[:], ih[:])
		copy(h.PeerID[:], t.id[:])
		// send handshake
		err = h.Send(c)
		if err == nil {
			// get response to handshake
			err = h.Recv(c)
			if err == nil {
				if bytes.Equal(ih[:], h.Infohash[:]) {
					// infohashes match
					pc := makePeerConn(c, t, h.PeerID)
					pc.start()
					t.onNewPeer(pc)
					t.cmtx.Lock()
					t.conns[a.String()] = pc
					t.cmtx.Unlock()
					return nil
				} else {
					log.Warn("Infohash missmatch")
				}
			}
		}
		log.Debugf("didn't complete handshake with peer: %s", err)
		// bad thing happened
		c.Close()
	}
	log.Debugf("didn't connect to %s: %s", a, err)
	return err
}

func (t *Torrent) broadcastHave(idx uint32) {
	msg := common.NewHave(idx)
	t.cmtx.Lock()
	for _, conn := range t.conns {
		go conn.Send(msg)
	}
	t.cmtx.Unlock()
}

// get metainfo for this torrent
func (t *Torrent) MetaInfo() *metainfo.TorrentFile {
	return t.st.MetaInfo()
}

func (t *Torrent) Name() string {
	return t.MetaInfo().TorrentName()
}

// gracefully close torrent and flush to disk
func (t *Torrent) Close() {
	chnl := t.piece
	t.piece = nil
	close(chnl)
	t.st.Flush()
}

// callback called when we get a new inbound peer
func (t *Torrent) onNewPeer(c *PeerConn) {
	log.Infof("New peer (%s) for %s", c.id.String(), t.st.Infohash().Hex())
	// send our bitfields to them
	c.Send(t.Bitfield().ToWireMessage())
}

// handle a piece request
func (t *Torrent) onPieceRequest(c *PeerConn, req *common.PieceRequest) {
	if t.piece != nil {
		t.piece <- pieceEvent{c, req}
	}
}

func (t *Torrent) Run() {
	log.Infof("%s running", t.Name())
	for {
		ev, ok := <-t.piece
		if !ok {
			log.Infof("%s torrent run exit", t.Name())
			// channel closed
			return
		}
		r := ev.r
		if r.Length > 0 {
			log.Debugf("%s asked for piece %d %d-%d", ev.c.id.String(), r.Index, r.Begin, r.Begin+r.Length)
			// TODO: cache common pieces (?)
			p, err := t.st.GetPiece(r)
			if p == nil {
				// we don't have the piece
				log.Infof("%s asked for a piece we don't have for %s: %s", ev.c.id.String(), t.Name(), err)
				// TODO: should we close here?
				ev.c.Close()
			} else {
				// have the piece, send it
				dl := len(p.Data)
				d := make([]byte, 8+dl)
				log.Debugf("piece %d datasize %d", p.Index, dl)
				binary.BigEndian.PutUint32(d, p.Index)
				binary.BigEndian.PutUint32(d[4:], p.Begin)
				copy(d[8:], p.Data[:])
				msg := common.NewWireMessage(common.Piece, d)
				ev.c.Send(msg)
			}
		} else {
			log.Infof("%s asked for a zero length piece", ev.c.id.String())
			// TODO: should we close here?
			ev.c.Close()
		}

	}
}

// implements client.Algorithm
func (t *Torrent) Done() bool {
	return t.Bitfield().Completed()
}

// implements client.Algorithm
func (t *Torrent) Choke(id common.PeerID) bool {
	// TODO: implement choking
	return false
}
