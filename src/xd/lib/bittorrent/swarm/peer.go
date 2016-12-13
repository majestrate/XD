package swarm

import (
	"net"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/log"
)

// a peer connection
type PeerConn struct {
	c              net.Conn
	id             common.PeerID
	t              *Torrent
	send           chan *bittorrent.WireMessage
	bf             *bittorrent.Bitfield
	peerChoke      bool
	peerInterested bool
	usChoke        bool
	usInterseted   bool
}

func makePeerConn(c net.Conn, t *Torrent, id common.PeerID) *PeerConn {
	p := new(PeerConn)
	p.c = c
	p.t = t
	p.peerChoke = true
	p.usChoke = true
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
	log.Debugf("got %d from %s", msg.Len(), c.id)
	return
}

func (c *PeerConn) HasPiece(piece int) bool {
	if c.bf == nil {
		// no bitfield
		return false
	}
	return c.bf.Has(piece)
}

// return true if this peer is choked otherwise return false
func (c *PeerConn) PeerChoked() bool {
	return c.peerChoke
}

func (c *PeerConn) remoteUnchoke() {
	if !c.peerChoke {
		log.Warnf("remote peer %s sent multiple unchokes", c.id.String())
	}
	c.peerChoke = false
}

func (c *PeerConn) remoteChoke() {
	if c.peerChoke {
		log.Warnf("remote peer %s sent multiple chokes", c.id.String())
	}
	c.peerChoke = true
}

func (c *PeerConn) markInterested() {
	c.peerInterested = true
}

func (c *PeerConn) markNotInterested() {
	c.peerInterested = false
}

// run read loop
func (c *PeerConn) runReader() {
	var err error
	for err == nil {
		var msg *bittorrent.WireMessage
		msg, err = c.Recv()
		if err == nil {
			if msg.KeepAlive() {
				log.Debugf("keepalive from %s", c.id)
				continue
			}
			msgid := msg.MessageID()
			log.Debugf("%s from %s", msgid.String(), c.id.String())
			if msgid == bittorrent.BitField {
				c.bf = bittorrent.NewBitfield(len(c.t.MetaInfo().Info.Pieces), msg.Payload())
				log.Debugf("got bitfield from %s", c.id.String())
				continue
			}
			if msgid == bittorrent.Choke {
				c.remoteChoke()
				continue
			}
			if msgid == bittorrent.UnChoke {
				c.remoteUnchoke()
				continue
			}
			if msgid == bittorrent.Interested {
				c.markInterested()
				continue
			}
			if msgid == bittorrent.NotInterested {
				c.markNotInterested()
				continue
			}
			if msgid == bittorrent.Request {
				ev := msg.GetPieceRequest()
				c.t.onPieceRequest(c, ev)
				continue
			}
		}
	}
	log.Errorf("%s read error: %s", c.id, err)
}

// run write loop
func (c *PeerConn) runWriter() {
	var err error
	for err == nil {
		select {
		case msg, ok := <-c.send:
			if ok {
				err = msg.Send(c.c)
			}
		}
	}
}
