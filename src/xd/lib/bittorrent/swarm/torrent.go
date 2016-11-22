package swarm

import (
	"xd/lib/bittorrent"
	"xd/lib/storage"
)

// an event triggered when we get an inbound wire message from a peer we are connected with on this torrent
type wireEvent struct {
	c *PeerConn
	msg *bittorrent.WireMessage
}

type Torrent struct {
	st storage.Torrent
	recv chan wireEvent
}

func (t *Torrent) OnWireMessage(c *PeerConn, msg *bittorrent.WireMessage) {
	t.recv <- wireEvent{c, msg}
}
