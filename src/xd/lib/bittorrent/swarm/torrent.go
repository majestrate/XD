package swarm

import (
	"bytes"
	"net"
	"sync"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/bittorrent/extensions"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/metainfo"
	"xd/lib/network"
	"xd/lib/storage"
	"xd/lib/tracker"
)

// single torrent tracked in a swarm
type Torrent struct {
	Net         network.Network
	Trackers    []tracker.Announcer
	announcer   *time.Ticker
	id          common.PeerID
	st          storage.Torrent
	piece       chan pieceEvent
	obconns     map[string]*PeerConn
	ibconns     map[string]*PeerConn
	mtx         sync.Mutex
	pt          *pieceTracker
	defaultOpts *extensions.ExtendedOptions
}

func newTorrent(st storage.Torrent) *Torrent {
	t := &Torrent{
		st:          st,
		piece:       make(chan pieceEvent),
		ibconns:     make(map[string]*PeerConn),
		obconns:     make(map[string]*PeerConn),
		defaultOpts: extensions.New(),
	}
	t.pt = createPieceTracker(st, t.getRarestPiece)
	t.pt.have = t.broadcastHave
	return t
}

func (t *Torrent) getRarestPiece(remote *bittorrent.Bitfield) (idx uint32) {
	var swarm []*bittorrent.Bitfield
	t.VisitPeers(func(c *PeerConn) {
		if c.bf != nil {
			swarm = append(swarm, c.bf)
		}
	})
	idx = remote.FindRarest(swarm)
	return
}

// call a visitor on each open peer connection
func (t *Torrent) VisitPeers(v func(*PeerConn)) {
	var conns []*PeerConn
	t.mtx.Lock()
	for _, conn := range t.obconns {
		if conn != nil {
			conns = append(conns, conn)
		}
	}
	for _, conn := range t.ibconns {
		if conn != nil {
			conns = append(conns, conn)
		}
	}
	t.mtx.Unlock()
	for _, conn := range conns {
		v(conn)
	}
}

func (t *Torrent) GetStatus() TorrentStatus {
	name := t.Name()
	var peers []*PeerConnStats
	t.VisitPeers(func(c *PeerConn) {
		peers = append(peers, c.Stats())
	})
	log.Debugf("unlocked torrent mutex for %s", name)
	state := Downloading
	if t.Done() {
		state = Seeding
	}
	return TorrentStatus{
		Peers:    peers,
		Name:     name,
		State:    state,
		Infohash: t.MetaInfo().Infohash().Hex(),
	}

}

func (t *Torrent) Bitfield() *bittorrent.Bitfield {
	return t.st.Bitfield()
}

// start annoucing on all trackers
func (t *Torrent) StartAnnouncing() {
	for _, tr := range t.Trackers {
		go t.Announce(tr, tracker.Started)
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
		go t.Announce(tr, tracker.Stopped)
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
				go t.Announce(tr, tracker.Nop)
			}
		}
	}
}

// do an announce
func (t *Torrent) Announce(tr tracker.Announcer, event tracker.Event) {
	req := &tracker.Request{
		Infohash: t.st.Infohash(),
		PeerID:   t.id,
		IP:       t.Net.Addr(),
		Port:     6881,
		Event:    event,
		NumWant:  10, // TODO: don't hardcode
		Left:     t.st.DownloadRemaining(),
	}
	resp, err := tr.Announce(req)
	if err == nil {
		for _, p := range resp.Peers {
			a, e := p.Resolve(t.Net)
			if e == nil {
				if a.String() == t.Net.Addr().String() {
					// don't connect to self or a duplicate
					continue
				}
				if t.HasOBConn(a) {
					continue
				}
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
		if t.HasIBConn(a) {
			return
		}
		if !t.HasOBConn(a) {
			err := t.AddPeer(a, id)
			if err == nil {
				triesLeft = 10
			} else {
				triesLeft--
			}
			if triesLeft == 0 {
				return
			}
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (t *Torrent) HasIBConn(a net.Addr) (has bool) {
	t.mtx.Lock()
	_, has = t.ibconns[a.String()]
	t.mtx.Unlock()
	return
}

func (t *Torrent) HasOBConn(a net.Addr) (has bool) {
	t.mtx.Lock()
	_, has = t.obconns[a.String()]
	t.mtx.Unlock()
	return
}

func (t *Torrent) addOBPeer(c *PeerConn) {
	t.mtx.Lock()
	t.obconns[c.c.RemoteAddr().String()] = c
	t.mtx.Unlock()
}

func (t *Torrent) removeOBConn(c *PeerConn) {
	t.mtx.Lock()
	delete(t.obconns, c.c.RemoteAddr().String())
	t.mtx.Unlock()
}

func (t *Torrent) addIBPeer(c *PeerConn) {
	t.mtx.Lock()
	t.ibconns[c.c.RemoteAddr().String()] = c
	t.mtx.Unlock()
	c.inbound = true
}

func (t *Torrent) removeIBConn(c *PeerConn) {
	t.mtx.Lock()
	delete(t.ibconns, c.c.RemoteAddr().String())
	t.mtx.Unlock()
}

// connect to a new peer for this swarm, blocks
func (t *Torrent) AddPeer(a net.Addr, id common.PeerID) error {
	c, err := t.Net.Dial(a.Network(), a.String())
	if err == nil {
		// connected
		ih := t.st.Infohash()
		// build handshake
		h := new(bittorrent.Handshake)
		// enable bittorrent extensions
		h.Reserved.Set(bittorrent.Extension)
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
					var opts *extensions.ExtendedOptions
					if h.Reserved.Has(bittorrent.Extension) {
						opts = t.defaultOpts.Copy()
					}
					pc := makePeerConn(c, t, h.PeerID, opts)
					pc.start()
					t.addOBPeer(pc)
					pc.Send(t.Bitfield().ToWireMessage())
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
	log.Infof("%s got piece %d", t.Name(), idx)
	conns := make(map[string]*PeerConn)
	t.mtx.Lock()
	for k, conn := range t.ibconns {
		if conn != nil {
			conns[k] = conn
		}
	}
	for k, conn := range t.obconns {
		if conn != nil {
			conns[k] = conn
		}
	}
	t.mtx.Unlock()
	for _, conn := range conns {
		go conn.Send(msg)
	}
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
	a := c.c.RemoteAddr()
	if t.HasIBConn(a) {
		log.Infof("duplicate peer from %s", a)
		c.Close()
		return
	}
	log.Infof("New peer (%s) for %s", c.id.String(), t.st.Infohash().Hex())
	t.addIBPeer(c)
	c.start()
	c.Send(t.Bitfield().ToWireMessage())
}

// handle a piece request
func (t *Torrent) onPieceRequest(c *PeerConn, req *common.PieceRequest) {
	if t.piece != nil {
		t.piece <- pieceEvent{c, req}
	}
}

func (t *Torrent) Run() {
	go t.handlePieces()
	for !t.Done() {
		time.Sleep(time.Minute)
	}
	log.Infof("%s is seeding", t.Name())
	for _, tr := range t.Trackers {
		go t.Announce(tr, "completed")
	}
}

func (t *Torrent) handlePieces() {
	log.Infof("%s running", t.Name())
	for {
		ev, ok := <-t.piece
		if !ok {
			log.Infof("%s torrent run exit", t.Name())
			// channel closed
			return
		}
		if ev.r.Length > 0 {
			log.Debugf("%s asked for piece %d %d-%d", ev.c.id.String(), ev.r.Index, ev.r.Begin, ev.r.Begin+ev.r.Length)
			// TODO: cache common pieces (?)
			err := t.st.VisitPiece(ev.r, func(p *common.PieceData) error {
				// have the piece, send it
				ev.c.Send(p.ToWireMessage())
				log.Debugf("%s queued piece %d %d-%d", ev.c.id.String(), ev.r.Index, ev.r.Begin, ev.r.Begin+ev.r.Length)
				return nil
			})
			if err != nil {
				ev.c.Close()
			}
		} else {
			log.Infof("%s asked for a zero length piece", ev.c.id.String())
			// TODO: should we close here?
			ev.c.Close()
		}

	}
}

func (t *Torrent) Done() bool {
	return t.Bitfield().Completed()
}
