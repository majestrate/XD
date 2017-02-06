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

// how big should we download pieces at a time (bytes)?
const BlockSize = 1024 * 16

const Missing = 0
const Pending = 1
const Obtained = 2

// an event triggered when we get an inbound wire message from a peer we are connected with on this torrent asking for a piece
type pieceEvent struct {
	c *PeerConn
	r *bittorrent.PieceRequest
}

// cached downloading piece
type cachedPiece struct {
	piece    *common.Piece
	progress []byte
	mtx      sync.RWMutex
}

// get unfilled available block offset
func (p *cachedPiece) nextOffset() (idx int) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	for idx < len(p.progress) {
		if p.progress[idx] == Missing {
			// mark progress as pending
			var i int
			for i < BlockSize {
				p.progress[idx+i] = Pending
				i++
			}
			return
		}
		idx += BlockSize
	}
	if idx >= len(p.progress) {
		// fail
		idx = -1
	}
	return
}

// is this piece done downloading ?
func (p *cachedPiece) done() bool {
	for _, b := range p.progress {
		if b != Obtained {
			return false
		}
	}
	return true
}

// put a slice of data at offset
func (p *cachedPiece) put(offset int, data []byte) {
	if offset+len(data) <= len(p.progress) {
		// put data
		copy(p.piece.Data[offset:], data)
		// put progress
		for idx, _ := range data {
			p.progress[idx+offset] = Obtained
		}
	} else {
		log.Warnf("block out of range %d", offset)
	}
}

// cancel a slice
func (p *cachedPiece) cancel(offset, length int) {
	if offset+length <= len(p.progress) {
		for length > 0 {
			length--
			p.progress[offset+length] = Missing
		}
	}
}

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
	// pending incomplete pieces and who is fetching them
	pending map[uint32]*PeerConn
	pmtx    sync.RWMutex
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
				if a.String() == t.Net.Addr().String() {
					// don't connect to self
					continue
				}
				// no error resolving
				go t.PersistPeer(a, p.ID)
			} else {
				log.Warnf("failed to resolve peer %s", e.Error())
			}
		}
	} else {
		log.Warnf("failed to announce to %s: %s", tr.Name(), err)
	}
}

// persit a connection to a peer
func (t *Torrent) PersistPeer(a net.Addr, id common.PeerID) {
	log.Debugf("persisting peer %s", id)
	for !t.Done() {
		t.AddPeer(a, id)
	}
}

// connect to a new peer for this swarm, blocks
func (t *Torrent) AddPeer(a net.Addr, id common.PeerID) {
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
					return
				}
			}
		}
		log.Warnf("didn't complete handshake with peer: %s", err)
		// bad thing happened
		c.Close()
		return
	}
	log.Infof("didn't connect to %s: %s", a, err)
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

func (t *Torrent) storePiece(p *common.Piece) {
	log.Infof("storing piece %d for %s", p.Index, t.st.Infohash().Hex())
	err := t.st.PutPiece(p)
	if err != nil {
		log.Errorf("failed to put piece for %s: %s", t.Name(), err)
	}
	t.pmtx.Lock()
	delete(t.pending, uint32(p.Index))
	t.pmtx.Unlock()
	t.st.Flush()
}

func (t *Torrent) cancelPiece(idx uint32) {
	t.pmtx.Lock()
	delete(t.pending, idx)
	t.pmtx.Unlock()
}

func (t *Torrent) markPieceInProgress(idx uint32, c *PeerConn) {
	t.pmtx.Lock()
	t.pending[idx] = c
	t.pmtx.Unlock()
}

func (t *Torrent) pieceRequested(idx uint32) bool {
	t.pmtx.Lock()
	_, ok := t.pending[idx]
	t.pmtx.Unlock()
	return ok
}

// callback called when we get a new inbound peer
func (t *Torrent) onNewPeer(c *PeerConn) {
	log.Infof("New peer (%s) for %s", c.id.String(), t.st.Infohash().Hex())
	// send our bitfields to them
	c.Send(t.Bitfield().ToWireMessage())
}

// handle a piece request
func (t *Torrent) onPieceRequest(c *PeerConn, req *bittorrent.PieceRequest) {
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
			// TODO: cache common pieces
			p := t.st.GetPiece(r.Index)
			if p == nil {
				// we don't have the piece
				log.Infof("%s asked for a piece we don't have for %s", ev.c.id.String(), t.Name())
				// TODO: should we close here?
				ev.c.Close()
			} else {
				// have the piece, send it
				d := make([]byte, 8+r.Length)
				binary.BigEndian.PutUint32(d, r.Index)
				binary.BigEndian.PutUint32(d[4:], r.Begin)
				copy(d[8:], p.Data[r.Begin:r.Begin+r.Length])
				msg := bittorrent.NewWireMessage(bittorrent.Piece, d)
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
