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
	bf bittorrent.Bitfield
	recv chan wireEvent
}

func (t *Torrent) OnNewPeer(c *PeerConn) {
	
}

func (t *Torrent) OnWireMessage(c *PeerConn, msg *bittorrent.WireMessage) {
	t.recv <- wireEvent{c, msg}
}

func (t *Torrent) Run() {
	for {
		ev, ok := <- t.recv
		if !ok {
			// channel closed
			return
		}
		if ev.msg.KeepAlive() {
			continue
		}
		id := ev.msg.MessageID()
		if id == 5 {
			// we got a bitfield
		}
	}
}
