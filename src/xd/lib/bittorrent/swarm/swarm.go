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
	net      network.Network
	Torrents Holder
	id       common.PeerID
}

// wait until we get a network context
func (sw *Swarm) WaitForNetwork() {
	for sw.net == nil {
		time.Sleep(time.Second)
	}
}

func (sw *Swarm) startTorrent(t *Torrent) {
	sw.WaitForNetwork()
	// give network
	t.Net = sw.net
	// give peerid
	t.id = sw.id
	// add trackers
	info := t.MetaInfo()
	for _, u := range info.GetAllAnnounceURLS() {
		tr := tracker.FromURL(sw.net, u)
		if tr != nil {
			t.Trackers = append(t.Trackers, tr)
		}
	}
	// start annoucing
	go t.StartAnnouncing()
	// handle messages
	go t.Run()
}

// blocking run of swarm
// start accepting inbound connections
func (sw *Swarm) Run() (err error) {
	sw.WaitForNetwork()
	log.Infof("swarm obtained network address: %s", sw.net.Addr())
	for err == nil {
		var c net.Conn
		c, err = sw.net.Accept()
		if err == nil {
			log.Debugf("got inbound bittorrent connection from %s", c.RemoteAddr())
			go sw.inboundConn(c)
		}
	}
	return
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
func (sw *Swarm) AddTorrent(t storage.Torrent) (err error) {
	sw.Torrents.addTorrent(t)
	tr := sw.Torrents.GetTorrent(t.Infohash())
	go sw.startTorrent(tr)
	return
}

// inject network context when it's ready
func (sw *Swarm) SetNetwork(net network.Network) {
	sw.net = net
}

// create a new swarm using a storage backend for storing downloads and torrent metadata
func NewSwarm(storage storage.Storage) *Swarm {
	sw := &Swarm{
		Torrents: Holder{
			st:       storage,
			torrents: make(map[string]*Torrent),
		},
	}
	sw.id = common.GeneratePeerID()
	log.Infof("generated peer id %s", sw.id.String())
	return sw
}
