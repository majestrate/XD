package swarm

import (
	"bytes"
	"errors"
	"github.com/majestrate/XD/lib/bittorrent"
	"github.com/majestrate/XD/lib/bittorrent/extensions"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/dht"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/metainfo"
	"github.com/majestrate/XD/lib/network"
	"github.com/majestrate/XD/lib/stats"
	"github.com/majestrate/XD/lib/storage"
	"github.com/majestrate/XD/lib/sync"
	"github.com/majestrate/XD/lib/tracker"
	"github.com/majestrate/XD/lib/util"
	"net"
	"time"
)

// max peers peer swarm default
const DefaultMaxSwarmPeers = 50

// rate name for upload
const RateUpload = "upload"

// rate name for download
const RateDownload = "download"

var defaultRates = []string{RateDownload, RateUpload}

// single torrent tracked in a swarm
type Torrent struct {
	TID              int64
	addr             net.Addr
	Completed        func()
	Started          func()
	Stopped          func()
	RemoveSelf       func()
	netacces         sync.Mutex
	suspended        bool
	Network          func() network.Network
	Trackers         map[string]tracker.Announcer
	announcers       map[string]*torrentAnnounce
	announceMtx      sync.Mutex
	announceTicker   *time.Ticker
	id               common.PeerID
	st               storage.Torrent
	obconns          map[string]*PeerConn
	ibconns          map[string]*PeerConn
	connMtx          sync.Mutex
	pt               *pieceTracker
	defaultOpts      extensions.Message
	closing          bool
	started          bool
	MaxRequests      int
	MaxPeers         uint
	pexState         PEXSwarmState
	xdht             *dht.XDHT
	statsTracker     *stats.Tracker
	tx               uint64
	rx               uint64
	seeding          bool
	metaInfo         []byte
	pendingInfoBF    *bittorrent.Bitfield
	requestingInfoBF *bittorrent.Bitfield
	puttingMetaInfo  bool
	addedAt          time.Time
	peersPool        sync.Pool
	lastPEX          time.Time
	pexInterval      time.Duration
}

func (t *Torrent) ShouldAcceptNewPeer() bool {
	state := t.GetStatus().State
	return state == Downloading || state == Seeding
}

func (t *Torrent) getNextPeer() *PeerConn {
	p := t.peersPool.Get()
	return p.(*PeerConn)
}

func (t *Torrent) DownloadDir() string {
	return t.st.DownloadDir()
}

func (t *Torrent) AddedAt() time.Time {
	return t.addedAt
}

func (t *Torrent) Ready() bool {
	return t.st.MetaInfo() != nil
}

// implements io.Closer
func (t *Torrent) Close() error {
	if t.closing {
		return nil
	}
	t.closing = true
	t.started = false
	t.VisitPeers(func(c *PeerConn) {
		c.Close()
	})
	t.saveStats()
	return t.st.Flush()
}

func (t *Torrent) shouldAnnounce(name string) bool {
	return time.Now().After(t.nextAnnounceFor(name))
}

func (t *Torrent) SetPieceWindow(n int) {
	t.MaxRequests = n
	t.VisitPeers(func(c *PeerConn) {
		c.MaxParalellRequests = n
	})
	// t.pt.maxPending = n
}

func (t *Torrent) nextAnnounceFor(name string) (tm time.Time) {
	t.announceMtx.Lock()
	a, ok := t.announcers[name]
	if ok {
		tm = a.next
	} else {
		tm = time.Now()
		t.announcers[name] = &torrentAnnounce{
			next:     tm,
			t:        t,
			announce: t.Trackers[name],
		}
	}
	t.announceMtx.Unlock()
	return tm
}

var tIDCounter = int64(0)

func newTorrent(st storage.Torrent, getNet func() network.Network) *Torrent {
	t := &Torrent{
		TID:          tIDCounter,
		Trackers:     make(map[string]tracker.Announcer),
		announcers:   make(map[string]*torrentAnnounce),
		st:           st,
		Network:      getNet,
		ibconns:      make(map[string]*PeerConn),
		obconns:      make(map[string]*PeerConn),
		MaxRequests:  DefaultMaxParallelRequests,
		MaxPeers:     DefaultMaxSwarmPeers,
		statsTracker: stats.NewTracker(),
		addedAt:      time.Now(),
		lastPEX:      time.Now(),
		pexInterval:  time.Minute * 2,
	}
	t.peersPool.New = func() interface{} { return &PeerConn{} }
	tIDCounter++
	for _, rate := range defaultRates {
		t.statsTracker.NewRate(rate)
	}
	if t.Ready() {
		bytes := t.st.MetaInfo().RawInfo
		t.defaultOpts = extensions.NewOur(uint32(len(bytes)))
		t.metaInfo = bytes
	} else {
		t.defaultOpts = extensions.NewOur(0)
	}
	// set default pex dialect supported
	t.defaultOpts.SetSupported(DefaultPEXDialect)
	// set ut_metadata supported
	t.defaultOpts.SetSupported(extensions.UTMetaData)
	t.pt = createPieceTracker(st, t.getRarestPiece)
	t.pt.have = t.broadcastHave
	return t
}

func (t *Torrent) getRarestPiece(remote *bittorrent.Bitfield, exclude []uint32) (idx uint32, has bool) {
	var swarm []*bittorrent.Bitfield
	t.VisitPeers(func(c *PeerConn) {
		if c.bf != nil {
			swarm = append(swarm, c.bf)
		}
	})
	m := make(map[uint32]bool)
	for idx := range exclude {
		m[exclude[idx]] = true
	}
	bt := t.st.Bitfield()
	idx, has = remote.FindRarest(swarm, func(idx uint32) bool {
		return bt.Has(idx) || m[idx]
	})
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
	t.connMtx.Lock()
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
	t.connMtx.Unlock()
	for _, conn := range conns {
		v(conn)
	}
}

func (t *Torrent) RX() (rx int64) {
	t.VisitPeers(func(c *PeerConn) {
		rx += int64(c.rx.Mean())
	})
	return
}

func (t *Torrent) TX() (tx int64) {
	t.VisitPeers(func(c *PeerConn) {
		tx += int64(c.tx.Mean())
	})
	return
}

func (t *Torrent) GetStatus() TorrentStatus {

	var addr string
	if t.addr != nil {
		addr = t.addr.String()
	}
	name := t.Name()
	var peers []*PeerConnStats
	t.VisitPeers(func(c *PeerConn) {
		peers = append(peers, c.Stats())
	})
	state := Downloading
	if t.st.Checking() {
		state = Checking
	}
	if !t.Ready() {
		return TorrentStatus{
			Peers:    peers,
			Name:     name,
			State:    state,
			Infohash: t.st.Infohash().Hex(),
			TX:       t.tx,
			RX:       t.rx,
			Us: PeerConnStats{
				TX:     float64(t.TX()),
				RX:     float64(t.RX()),
				ID:     t.id.String(),
				Client: util.ClientNameFromID(t.id[:]),
				Addr:   addr,
			},
		}
	}
	if t.Done() {
		state = Seeding
	} else if t.closing || !t.started {
		state = Stopped
	}
	if t.st.Checking() {
		state = Checking
	}

	bf := t.Bitfield()
	var files []TorrentFileInfo
	nfo := t.st.MetaInfo().Info
	var idx uint64
	f := nfo.GetFiles()
	if len(f) == 1 {
		b := bittorrent.Bitfield{
			Data:   bf.Data,
			Length: bf.Length,
		}
		files = append(files, TorrentFileInfo{
			FileInfo: f[0],
			Progress: b.Progress(),
		})
	} else {
		for _, file := range f {
			l := file.Length / uint64(nfo.PieceLength)
			// XXX: this below here is wrong because how the bits are packed in the bitfield
			l /= 8
			plen := l
			var data []byte
			if l == 0 {
				data = []byte{bf.Data[idx]}
				plen = 1
			} else if idx+l < uint64(len(bf.Data)) {
				data = bf.Data[idx : idx+l]
			} else {
				data = bf.Data[idx:]
			}
			b := bittorrent.Bitfield{
				Data:   data,
				Length: uint32(plen),
			}
			files = append(files, TorrentFileInfo{
				FileInfo: file,
				Progress: b.Progress(),
			})
			idx += l
		}
	}
	b := bittorrent.Bitfield{
		Data:   bf.Data,
		Length: bf.Length,
	}
	return TorrentStatus{
		Peers:    peers,
		Name:     name,
		State:    state,
		Infohash: t.MetaInfo().Infohash().Hex(),
		Progress: b.Progress(),
		Files:    files,
		TX:       t.tx,
		RX:       t.rx,
		Us: PeerConnStats{
			TX:     float64(t.TX()),
			RX:     float64(t.RX()),
			ID:     t.id.String(),
			Client: util.ClientNameFromID(t.id[:]),
			Addr:   addr,
		},
	}
}

func (t *Torrent) Bitfield() *bittorrent.Bitfield {
	return t.st.Bitfield()
}

// manually announce as seed to all trackers
// blocks until done
func (t *Torrent) AnnounceSeed() {
	var wg sync.WaitGroup
	for name := range t.Trackers {
		wg.Add(1)
		go func() {
			t.announce(name, tracker.Completed)
			wg.Add(-1)
		}()
	}
	wg.Wait()
}

// start annoucing on all trackers
func (t *Torrent) StartAnnouncing() {
	// wait for network
	t.addr = t.Network().Addr()
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
func (t *Torrent) StopAnnouncing(announce bool) {
	if t.announceTicker != nil {
		t.announceTicker.Stop()
		t.announceTicker = nil
	}
	if announce {
		var wg sync.WaitGroup
		for n := range t.Trackers {
			wg.Add(1)
			go func(name string) {
				log.Debugf("%s stopping", name)
				t.announce(name, tracker.Stopped)
				log.Debugf("%s stopped", name)
				wg.Add(-1)
			}(n)
		}
		wg.Wait()
	}
}

// poll announce ticker channel and issue announces
func (t *Torrent) pollAnnounce() {
	for t.announceTicker != nil {
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
				t.announce(name, ev)
			}
		}
	}
}

func (t *Torrent) announce(name string, ev tracker.Event) {
	t.announceMtx.Lock()
	a := t.announcers[name]
	t.announceMtx.Unlock()
	if a != nil {
		err := a.tryAnnounce(ev)
		if err == nil {
			a.fails = 0
		} else {
			log.Warnf("announce to %s failed: %s", name, err)
			a.fails++
		}
	}
}

// add peers to torrent
func (t *Torrent) addPeers(peers []common.Peer) {
	for _, p := range peers {
		if !t.NeedsPeers() {
			// no more peers needed
			return
		}
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
	for !t.closing {
		if t.HasIBConn(a) {
			return
		}
		if !t.HasOBConn(a) {
			err := t.DialPeer(a, id)
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
	t.connMtx.Lock()
	_, has = t.ibconns[a.String()]
	t.connMtx.Unlock()
	return
}

func (t *Torrent) HasOBConn(a net.Addr) (has bool) {
	t.connMtx.Lock()
	_, has = t.obconns[a.String()]
	t.connMtx.Unlock()
	return
}

func (t *Torrent) addOBPeer(c *PeerConn) {
	addr := c.c.RemoteAddr()
	t.connMtx.Lock()
	t.obconns[addr.String()] = c
	t.connMtx.Unlock()
	t.pexState.onNewPeer(addr)
}

func (t *Torrent) removeOBConn(c *PeerConn) {
	addr := c.c.RemoteAddr()
	t.connMtx.Lock()
	delete(t.obconns, addr.String())
	t.connMtx.Unlock()
	t.pexState.onPeerDisconnected(addr)
}

func (t *Torrent) addIBPeer(c *PeerConn) {
	addr := c.c.RemoteAddr()
	t.connMtx.Lock()
	t.ibconns[addr.String()] = c
	t.connMtx.Unlock()
	c.inbound = true
	t.pexState.onNewPeer(addr)
}

func (t *Torrent) removeIBConn(c *PeerConn) {
	addr := c.c.RemoteAddr()
	t.connMtx.Lock()
	delete(t.ibconns, addr.String())
	t.connMtx.Unlock()
	t.pexState.onPeerDisconnected(addr)
}

func (t *Torrent) hasAllPendingInfo() bool {
	return t.pendingInfoBF.Completed()
}

func (t *Torrent) getMetaInfo() []byte {
	if t.metaInfo == nil {
		info := t.st.MetaInfo()
		if info != nil {
			t.metaInfo = info.RawInfo
		}
	}
	return t.metaInfo
}

func (t *Torrent) resetPendingInfo() {
	t.requestingInfoBF = bittorrent.NewBitfield(t.requestingInfoBF.Length, nil)
	t.pendingInfoBF = bittorrent.NewBitfield(t.pendingInfoBF.Length, nil)
	t.metaInfo = make([]byte, len(t.metaInfo))
	t.askAllMetadata()
}

func (t *Torrent) askAllMetadata() {
	t.VisitPeers(func(c *PeerConn) {
		if c.theirOpts.MetaData() {
			id, ok := c.theirOpts.Extensions[extensions.UTMetaData.String()]
			if ok {
				c.askNextMetadata(uint8(id))
			}
		}
	})
}

func (t *Torrent) putInfoSlice(idx uint32, data []byte) {
	if t.puttingMetaInfo {
		return
	}
	if t.metaInfo != nil && !t.Ready() {
		log.Debugf("put info slice idx=%d len=%d", idx, len(data))
		t.pendingInfoBF.Set(idx)
		copy(t.metaInfo[idx*(16*1024):], data)
		if t.hasAllPendingInfo() {
			t.puttingMetaInfo = true
			log.Debugf("got all info slices: %q", t.metaInfo)
			log.Info("putting metainfo")
			err := t.st.PutInfoBytes(t.metaInfo)
			if err == nil {
				// reset
				sz := uint32(len(t.metaInfo))
				t.defaultOpts.MetainfoSize = &sz
				t.VisitPeers(func(p *PeerConn) {
					p.Close()
				})
			} else {
				t.puttingMetaInfo = false
				log.Errorf("failed to get meta info: %s", err.Error())
				t.resetPendingInfo()
			}
		} else {
			log.Debug("need more info slices")
		}
	} else {
		log.Debug("unwarrented metainfo slice")
	}
}

func (t *Torrent) nextMetaInfoReq() *uint32 {
	if t.Ready() {
		return nil
	}
	if t.metaInfo == nil || t.pendingInfoBF == nil || t.requestingInfoBF == nil {
		log.Debug("no bitfield or metainfo")
		return nil
	}
	var i uint32
	for i < uint32(len(t.metaInfo)/(1024*16))+1 {
		if (!t.pendingInfoBF.Has(i)) && (!t.requestingInfoBF.Has(i)) {
			t.requestingInfoBF.Set(i)
			return &i
		}
		i++
	}
	return nil
}

// connect to a new peer for this swarm, blocks
func (t *Torrent) DialPeer(a net.Addr, id common.PeerID) error {
	if t.HasOBConn(a) {
		return nil
	}
	ih := t.st.Infohash()
	log.Debugf("%s %s ", a.String(), a.Network())
	c, err := t.Network().Dial(a.Network(), a.String())
	if err == nil {
		// connected
		// build handshake
		var h bittorrent.Handshake
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
					var opts extensions.Message
					if h.Reserved.Has(bittorrent.Extension) {
						opts = t.defaultOpts.Copy()
					}
					pc := makePeerConn(c, t, h.PeerID, opts)
					t.addOBPeer(pc)
					pc.start()
					if t.Ready() {
						pc.Send(t.Bitfield().ToWireMessage())
					}
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
	log.Debugf("%s got piece %d", t.Name(), idx)
	conns := make(map[string]*PeerConn)
	t.VisitPeers(func(c *PeerConn) {
		conns[c.c.RemoteAddr().String()] = c
	})
	for _, conn := range conns {
		conn.Send(msg)
	}
}

// get metainfo for this torrent
func (t *Torrent) MetaInfo() *metainfo.TorrentFile {
	return t.st.MetaInfo()
}

func (t *Torrent) Name() string {
	if t.Ready() {
		return t.MetaInfo().TorrentName()
	}
	return t.Infohash().Hex()
}

// return false if we reached max peers for this torrent
func (t *Torrent) NeedsPeers() bool {
	return t.NumPeers() <= t.MaxPeers
}

// callback called when we get a new inbound peer
func (t *Torrent) onNewPeer(c *PeerConn) {
	a := c.c.RemoteAddr()
	if t.HasIBConn(a) {
		log.Debugf("duplicate peer from %s", a)
		c.Close()
		return
	}
	if t.NeedsPeers() && t.Ready() {
		log.Debugf("New peer (%s) for %s", c.id.String(), t.st.Infohash().Hex())
		t.addIBPeer(c)
		c.start()
		c.Send(t.Bitfield().ToWireMessage())
	} else {
		c.Close()
	}
}

func (t *Torrent) Infohash() common.Infohash {
	return t.st.Infohash()
}

func (t *Torrent) run() {
	if t.Started != nil {
		go t.Started()
	}
	t.started = true
	go t.runRateTicker()
	counter := 0
	for !t.closing {
		if !t.Ready() {
			time.Sleep(time.Second)
			// reset pending info if we can't fetch it fast enough
			counter++
			if t.Ready() {
				continue
			} else if counter%30 == 0 && !t.puttingMetaInfo && t.requestingInfoBF != nil {
				// reset requesting info if we can't fetch it fast enough
				t.requestingInfoBF = bittorrent.NewBitfield(t.requestingInfoBF.Length, nil)
				t.askAllMetadata()
			}
			continue
		}
		if t.Done() {
			if t.seeding {
				break
			} else {
				var err error
				t.seeding, err = t.st.Seed()
				if t.seeding {
					log.Infof("%s is seeding", t.Name())
					t.AnnounceSeed()
				} else if err != nil {
					log.Errorf("failed to begin seeding: %s", err.Error())
				} else {
					log.Infof("will need to redownload pieces for %s", t.Name())
				}
			}
		}
		time.Sleep(time.Second)
	}
}

func (t *Torrent) Private() bool {
	info := t.MetaInfo()
	if info == nil {
		return false
	}
	return info.IsPrivate()
}

func (t *Torrent) tick() {

	if !t.Ready() {
		return
	}

	if !t.Private() {
		now := time.Now()
		if now.Sub(t.lastPEX) > t.pexInterval {
			la := t.Network().Addr()
			if la.Network() == "i2p" {
				connected, disconnected := t.pexState.PopDestHashLists()
				t.VisitPeers(func(p *PeerConn) {
					if p.SupportsI2PPEX() {
						p.sendI2PPEX(connected, disconnected)
					}
				})
			} else {
				var connected []common.Peer
				t.VisitPeers(func(p *PeerConn) {
					if len(connected) < 15 {
						connected = append(connected, p.btPeer())
					}
				})
				t.VisitPeers(func(p *PeerConn) {
					if p.SupportsLNPEX() {
						p.sendLNPEX(connected, []common.Peer{})
					}
				})
			}
			t.lastPEX = now
		}
	}

	if t.Done() {
		return
	}
	// expire and cancel all timed out pieces
	t.pt.iterCached(func(cp *cachedPiece) {
		if cp.isExpired() {
			if cp.pending.CountSet() > 0 {
				t.VisitPeers(func(conn *PeerConn) {
					conn.cancelPiece(cp.index)
				})
				cp.pending.Zero()
				log.Debugf("Expired piece %d with no recent activity for torrent: %s", cp.index, t.Name())
			}
			cp.lastActive = time.Now()
		}
	})
	t.VisitPeers(func(conn *PeerConn) {
		conn.tickDownload()
	})
}

func (t *Torrent) handlePieceRequest(c *PeerConn, r *common.PieceRequest) {

	if r.Length > 0 {
		var pc common.PieceData
		log.Debugf("%s asked for piece %d %d-%d", c.id.String(), r.Index, r.Begin, r.Begin+r.Length)
		if r.Length <= uint32(cap(c.sendPieceBuff)) {
			pc.Data = c.sendPieceBuff[:r.Length]
			err := t.st.GetPiece(*r, &pc)
			if err == nil {
				// have the piece, send it
				c.Send(pc.ToWireMessage())
				log.Debugf("%s queued piece %d %d-%d", c.id.String(), r.Index, r.Begin, r.Begin+r.Length)
			} else {
				c.Close()
			}
		} else {
			log.Infof("%s asked for oversized piece bytes=%d", c.id.String(), r.Length)
			c.Close()
		}
	} else {
		log.Infof("%s asked for a zero length piece", c.id.String())
		// TODO: should we close here?
		c.Close()
	}

}

func (t *Torrent) Done() bool {
	bf := t.Bitfield()
	if bf == nil {
		return false
	}
	return bf.Completed()
}

var ErrAlreadyStopped = errors.New("torrent already stopped")
var ErrAlreadyStarted = errors.New("torrent already started")

func (t *Torrent) runRateTicker() {
	for t.started {
		time.Sleep(time.Second)
		t.tx += t.statsTracker.Rate(RateUpload).Current()
		t.rx += t.statsTracker.Rate(RateDownload).Current()
		t.statsTracker.Tick()
	}
}

func (t *Torrent) Stop() error {
	if t.closing {
		return ErrAlreadyStopped
	}
	log.Info("stopping...")
	err := t.Close()
	log.Info("stopping announce")
	t.StopAnnouncing(true)
	log.Info("stoped announce...")
	if t.Stopped != nil {
		t.Stopped()
	}
	t.RemoveSelf()
	log.Info("stopped")
	return err
}

func (t *Torrent) Delete() error {
	t.Close()
	t.StopAnnouncing(true)
	err := t.st.Delete()
	if err == nil {
		t.RemoveSelf()
	}
	return err
}

func (t *Torrent) Remove() error {
	err := t.Stop()
	if err != nil {
		return err
	}
	t.RemoveSelf()
	return nil
}

func (t *Torrent) Start() error {
	if t.started {
		return ErrAlreadyStarted
	}
	t.closing = false
	t.StartAnnouncing()
	go t.run()
	return nil
}

func (t *Torrent) saveStats() (err error) {
	err = t.st.SaveStats(t.statsTracker)
	return
}
