package swarm

import (
	"net"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/network"
	"xd/lib/storage"
)

type Swarm struct {
	net network.Network
	Torrents *Holder
	id common.PeerID
}

func (sw *Swarm) Run() (err error) {
	for err == nil {
		var c net.Conn
		c, err = sw.net.Accept()
		if err == nil {
			go sw.inboundConn(c)
		}
	}
	return
}


// got inbound connection
func (sw *Swarm) inboundConn(c net.Conn) {
	h := new(bittorrent.Handshake)
	err := h.Recv(c)
	if err != nil {
		// read error
		c.Close()
		return
	}
	t := sw.Torrents.GetTorrent(h.Infohash)
	if t == nil {
		// no such torrent
		c.Close()
		return
	}

	// make peer conn
	p := makePeerConn(c, t, h.PeerID)
	
	// reply to handshake
	copy(h.PeerID[:], sw.id[:])
	err = h.Send(c)
	if err != nil {
		// write error
		c.Close()
		return
	}
	go p.runWriter()
	go p.runReader()
	t.OnNewPeer(p)
}

func (sw *Swarm) AddTorrents() (err error) {
	var ts []storage.Torrent
	ts, err = sw.Torrents.st.OpenAllTorrents()
	if err == nil {
		for _, t := range ts {
			sw.Torrents.addTorrent(t)
		}
	}
	return
}

func NewSwarm(storage storage.Storage, net network.Network) *Swarm {
	sw := &Swarm{
		net: net,
		Torrents: &Holder{
			st: storage,
			torrents: make(map[common.Infohash]*Torrent),
		},
	}
	sw.id.Generate()
	return sw
}
