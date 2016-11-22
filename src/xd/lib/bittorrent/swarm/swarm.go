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
			// get handshake
			go sw.recvHandshake(c)
		}
	}
	return
}


// read bittorrent handshake
func (sw *Swarm) recvHandshake(c net.Conn) {
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
	// reply
	copy(h.PeerID[:], sw.id[:])
	err = h.Send(c)
	if err != nil {
		// write error
		c.Close()
		return
	}
	
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
