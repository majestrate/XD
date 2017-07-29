package swarm

import (
	"net"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/bittorrent/extensions"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/network"
	"xd/lib/storage"
	"xd/lib/tracker"
)

// a bittorrent swarm tracking many torrents
type Swarm struct {
	closing  bool
	net      network.Network
	Torrents Holder
	id       common.PeerID
	trackers map[string]tracker.Announcer
}

func (sw *Swarm) Running() bool {
	return !sw.closing
}

// wait until we get a network context
func (sw *Swarm) WaitForNetwork() {
	for sw.net == nil {
		time.Sleep(time.Second)
	}
}

func (sw *Swarm) startTorrent(t *Torrent) {
	sw.WaitForNetwork()
	// give peerid
	t.id = sw.id
	// add open trackers
	for name := range sw.trackers {
		t.Trackers[name] = sw.trackers[name]
	}

	info := t.MetaInfo()
	for _, u := range info.GetAllAnnounceURLS() {
		tr := tracker.FromURL(u)
		if tr != nil {
			name := tr.Name()
			_, ok := t.Trackers[name]
			if !ok {
				t.Trackers[name] = tr
			}
		}
	}

	// start annoucing
	go t.StartAnnouncing()
	// handle messages
	go t.Run()
}

// got inbound connection
func (sw *Swarm) inboundConn(c net.Conn) {
	h := new(bittorrent.Handshake)
	log.Debug("read bittorrent handshake")
	err := h.Recv(c)
	if err != nil {
		log.Warn("read bittorrent handshake failed, closing connection")
		// read error
		c.Close()
		return
	}
	t := sw.Torrents.GetTorrent(h.Infohash)
	if t == nil {
		log.Warnf("we don't have torrent with infohash %s, closing connection", h.Infohash.Hex())
		// no such torrent
		c.Close()
		return
	}
	var opts *extensions.ExtendedOptions
	if h.Reserved.Has(bittorrent.Extension) {
		opts = extensions.New()
	}
	// reply to handshake
	copy(h.PeerID[:], sw.id[:])
	err = h.Send(c)

	if err != nil {
		log.Warnf("didn't send bittorrent handshake reply: %s, closing connection", err)
		// write error
		c.Close()
		return
	}
	// make peer conn
	p := makePeerConn(c, t, h.PeerID, opts)

	go p.runWriter()
	go p.runReader()
	t.onNewPeer(p)
}

// add a torrent to this swarm
func (sw *Swarm) AddTorrent(t storage.Torrent, fresh bool) (err error) {
	if fresh {
		t.VerifyAll(true)
	}
	sw.Torrents.addTorrent(t)
	tr := sw.Torrents.GetTorrent(t.Infohash())
	go sw.startTorrent(tr)
	return
}

// run with network context
func (sw *Swarm) Run(n network.Network) (err error) {
	// resolve open trackers

	// give network to torrents
	sw.Torrents.ForEachTorrent(func(t *Torrent) {
		t.ObtainedNetwork(n)
	})
	// give network to swarm
	sw.net = n
	log.Debug("gave network context to torrents")
	// accept inbound connections
	for err == nil {
		var c net.Conn
		c, err = n.Accept()
		if err == nil {
			log.Debugf("got inbound bittorrent connection from %s", c.RemoteAddr())
			go sw.inboundConn(c)
		}
	}
	if sw.Running() {
		// suspend torrent's network on abbrupt break
		sw.Torrents.ForEachTorrent(func(t *Torrent) {
			t.LostNetwork()
		})
	}
	sw.net = nil
	return
}

// create a new swarm using a storage backend for storing downloads and torrent metadata
func NewSwarm(storage storage.Storage) *Swarm {
	sw := &Swarm{
		Torrents: Holder{
			st:       storage,
			torrents: make(map[string]*Torrent),
		},
		trackers: map[string]tracker.Announcer{},
	}
	sw.id = common.GeneratePeerID()
	log.Infof("generated peer id %s", sw.id.String())
	return sw
}

// AddOpenTracker adds an opentracker by url to be used by this swarm
func (sw *Swarm) AddOpenTracker(url string) {
	tr := tracker.FromURL(url)
	if tr != nil {
		name := tr.Name()
		_, ok := sw.trackers[name]
		if !ok {
			sw.trackers[name] = tr
		}
	}

}

// implements io.Closer
func (sw *Swarm) Close() (err error) {
	if !sw.closing {
		sw.closing = true
		err = sw.Torrents.Close()
	}
	return
}
