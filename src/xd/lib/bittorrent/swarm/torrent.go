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
	Completed      func()
	Started        func()
	Stopped        func()
	netacces       sync.Mutex
	suspended      bool
	netContext     network.Network
	Trackers       map[string]tracker.Announcer
	announcers     map[string]*torrentAnnounce
	announceMtx    sync.Mutex
	announceTicker *time.Ticker
	id             common.PeerID
	st             storage.Torrent
	piece          chan pieceEvent
	obconns        map[string]*PeerConn
	ibconns        map[string]*PeerConn
	mtx            sync.Mutex
	pt             *pieceTracker
	defaultOpts    *extensions.ExtendedOptions
	closing        bool
}

func (t *Torrent) ObtainedNetwork(n network.Network) {
	t.netContext = n
	if t.suspended {
		t.suspended = false
		t.netacces.Unlock()
	}
}

func (t *Torrent) WaitForNetwork() {
	for t.netContext == nil {
		time.Sleep(time.Millisecond)
	}
}

// get our current network context
func (t *Torrent) Network() (n network.Network) {
	for t.suspended {
		time.Sleep(time.Millisecond)
	}
	t.netacces.Lock()
	n = t.netContext
	t.netacces.Unlock()
	return
}

// called when we lost network access abruptly
func (t *Torrent) LostNetwork() {
	if t.suspended {
		return
	}
	t.netacces.Lock()
	t.suspended = true
	t.netContext = nil
}

// implements io.Closer
func (t *Torrent) Close() error {
	if t.closing {
		return nil
	}
	chnl := t.piece
	t.piece = nil
	t.closing = true
	t.StopAnnouncing()
	t.VisitPeers(func(c *PeerConn) {
		c.Close()
	})
	for t.NumPeers() > 0 {
		time.Sleep(time.Millisecond)
	}
	close(chnl)
	return t.st.Flush()
}

func (t *Torrent) shouldAnnounce(name string) bool {
	return time.Now().After(t.nextAnnounceFor(name))
}

func (t *Torrent) nextAnnounceFor(name string) (tm time.Time) {
	t.announceMtx.Lock()
	a, ok := t.announcers[name]
	if ok {
		tm = a.next
	} else {
		tm = time.Now().Add(time.Minute)
		t.announcers[name] = &torrentAnnounce{
			next:     tm,
			t:        t,
			announce: t.Trackers[name],
		}
	}
	t.announceMtx.Unlock()
	return tm
}

func newTorrent(st storage.Torrent) *Torrent {
	t := &Torrent{
		Trackers:    make(map[string]tracker.Announcer),
		announcers:  make(map[string]*torrentAnnounce),
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

// NumPeers counts how many peers we have on this torrent
func (t *Torrent) NumPeers() (count uint) {
	t.VisitPeers(func(_ *PeerConn) {
		count++
	})
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
	t.WaitForNetwork()
	ev := tracker.Started
	if t.Done() {
		ev = tracker.Completed
	}
	for name := range t.Trackers {
		t.nextAnnounceFor(name)
		go t.announce(name, ev)
	}
	if t.announceTicker == nil {
		t.announceTicker = time.NewTicker(time.Second)
	}
	go t.pollAnnounce()
}

// stop annoucing on all trackers
func (t *Torrent) StopAnnouncing() {
	if t.announceTicker != nil {
		t.announceTicker.Stop()
	}
	for name := range t.Trackers {
		t.announce(name, tracker.Stopped)
	}
}

// poll announce ticker channel and issue announces
func (t *Torrent) pollAnnounce() {
	for {
		_, ok := <-t.announceTicker.C
		if !ok {
			// done
			return
		}
		ev := tracker.Nop
		if t.Done() {
			ev = tracker.Completed
		}
		for name := range t.Trackers {
			if t.shouldAnnounce(name) {
				go t.announce(name, ev)
			}
		}
	}
}

func (t *Torrent) announce(name string, ev tracker.Event) {
	t.announceMtx.Lock()
	log.Infof("announcing to %s", name)
	a := t.announcers[name]
	t.announceMtx.Unlock()
	err := a.tryAnnounce(ev)
	if err != nil {
		log.Warnf("announce to %s failed: %s", name, err)
	}
}

// add peers to torrent
func (t *Torrent) addPeers(peers []common.Peer) {
	for _, p := range peers {
		a, e := p.Resolve(t.Network())
		if e == nil {
			if a.String() == t.Network().Addr().String() {
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
				return
			} else {
				triesLeft--
			}
			if triesLeft <= 0 {
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
	if t.HasOBConn(a) {
		return nil
	}
	c, err := t.Network().Dial(a.Network(), a.String())
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
					t.addOBPeer(pc)
					pc.Send(t.Bitfield().ToWireMessage())
					pc.start()
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
	c.Send(t.Bitfield().ToWireMessage())
	c.start()

}

// handle a piece request
func (t *Torrent) onPieceRequest(c *PeerConn, req *common.PieceRequest) {
	if t.piece != nil {
		t.piece <- pieceEvent{c, req}
	}
}

func (t *Torrent) Run() {
	go t.handlePieces()
	if t.Started != nil {
		t.Started()
	}
	for !t.Done() {
		time.Sleep(time.Minute)
	}
	if t.Completed != nil {
		t.Completed()
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
		if ev.r != nil && ev.r.Length > 0 {
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
