package swarm

import (
	"net"
	"xd/lib/bittorrent"
	"xd/lib/common"
)

// a peer connection
type PeerConn struct {
	c net.Conn
	id common.PeerID
	t *Torrent
	send chan *bittorrent.WireMessage
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

// run read loop
func (c *PeerConn) runReader() {
	var err error
	for err == nil {
		msg := new(bittorrent.WireMessage)
		err = msg.Recv(c.c)
		if err == nil {
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
