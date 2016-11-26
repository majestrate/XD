package swarm

import (
	"net"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/log"
)

// a peer connection
type PeerConn struct {
	c net.Conn
	id common.PeerID
	t *Torrent
	send chan *bittorrent.WireMessage
	bf *bittorrent.Bitfield
}

func makePeerConn(c net.Conn, t *Torrent, id common.PeerID) *PeerConn {
	p := new(PeerConn)
	p.c = c
	p.t = t
	copy(p.id[:], id[:])
	p.send = make(chan *bittorrent.WireMessage, 8)
	return p
}

// send a bittorrent wire message to this peer
func (c *PeerConn) Send(msg *bittorrent.WireMessage) {
	c.send <- msg
}

// recv a bittorrent wire message (blocking)
func (c *PeerConn) Recv() (msg *bittorrent.WireMessage, err error) {
	msg = new(bittorrent.WireMessage)
	err = msg.Recv(c.c)
	return
}

func (c *PeerConn) HasPiece(piece int) bool {
	if c.bf == nil {
		// no bitfield
		return false
	}
	return c.bf.Has(piece)
}

// run read loop
func (c *PeerConn) runReader() {
	var err error
	for err == nil {
		var msg *bittorrent.WireMessage
		msg, err = c.Recv()
		if err == nil {
			if msg.KeepAlive() {
				continue
			}
			if msg.MessageID() == bittorrent.BITFIELD {
				c.bf = bittorrent.NewBitfield(len(c.t.MetaInfo().Info.Pieces), msg.Payload())
				log.Debugf("got bitfield from %s", c.id.String())
				continue
			}
			c.t.OnWireMessage(c, msg)
		}
	}
}


// run write loop
func (c *PeerConn) runWriter() {
	var err error
	for err == nil {
		select {
		case msg, ok := <- c.send:
			if ok {
				err = msg.Send(c.c)
			}
		}
	}
}
