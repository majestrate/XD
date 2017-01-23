package swarm

import (
	"io"
	"net"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/bittorrent/client"
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
	// request algorithm
	Algorithm client.Algorithm
	// done callback
	Done func()
}

func makePeerConn(c net.Conn, t *Torrent, id common.PeerID) *PeerConn {
	p := new(PeerConn)
	p.c = c
	p.t = t
	p.peerChoke = true
	p.usChoke = true
	copy(p.id[:], id[:])
	p.send = make(chan *bittorrent.WireMessage, 8)
	p.Algorithm = t
	return p
}

// send a bittorrent wire message to this peer
func (c *PeerConn) Send(msg *bittorrent.WireMessage) {
	if c.send == nil {
		return
	}
	c.send <- msg
}

// recv a bittorrent wire message (blocking)
func (c *PeerConn) Recv() (msg *bittorrent.WireMessage, err error) {
	msg = new(bittorrent.WireMessage)
	err = msg.Recv(c.c)
	log.Debugf("got %d from %s", msg.Len(), c.id)
	return
}

// send choke
func (c *PeerConn) Choke() {
	if !c.usChoke {
		log.Debugf("choke peer %s", c.id.String())
		c.Send(bittorrent.NewWireMessage(bittorrent.Choke, nil))
		c.usChoke = true
	}
}

// send unchoke
func (c *PeerConn) Unchoke() {
	if c.usChoke {
		log.Debugf("unchoke peer %s", c.id.String())
		c.Send(bittorrent.NewWireMessage(bittorrent.UnChoke, nil))
		c.usChoke = false
	}
}

func (c *PeerConn) HasPiece(piece int) bool {
	if c.bf == nil {
		// no bitfield
		return false
	}
	return c.bf.Has(piece)
}

// return true if this peer is choking us otherwise return false
func (c *PeerConn) RemoteChoking() bool {
	return c.peerChoke
}

// return true if we are choking the remote peer otherwise return false
func (c *PeerConn) Chocking() bool {
	return c.usChoke
}

func (c *PeerConn) remoteUnchoke() {
	if !c.peerChoke {
		log.Warnf("remote peer %s sent multiple unchokes", c.id.String())
	}
	c.peerChoke = false
	log.Debugf("%s unchoked us", c.id.String())
}

func (c *PeerConn) remoteChoke() {
	if c.peerChoke {
		log.Warnf("remote peer %s sent multiple chokes", c.id.String())
	}
	c.peerChoke = true
	log.Debugf("%s choked us", c.id.String())
}

func (c *PeerConn) markInterested() {
	c.peerInterested = true
	log.Debugf("%s is interested", c.id.String())
}

func (c *PeerConn) markNotInterested() {
	c.peerInterested = false
	log.Debugf("%s is not interested", c.id.String())
}

func (c *PeerConn) Close() {
	log.Debugf("%s closing connection", c.id.String())
	chnl := c.send
	c.send = nil
	close(chnl)
	c.c.Close()
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
			if msgid == bittorrent.Piece {
				c.t.gotPieceData(msg.GetPieceData())
			}
		}
	}
	if err != io.EOF {
		log.Errorf("%s read error: %s", c.id, err)
	}
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

// run download loop
func (c *PeerConn) runDownload() {
	for c.Algorithm != nil && !c.Algorithm.Done() {
		// check for choke
		if c.Algorithm.Choke(c.id) {
			c.Choke()
			continue
		} else {
			c.Unchoke()
		}
		for c.RemoteChoking() {
			// wait until we are unchoked
			log.Debugf("%s waiting for unchoke", c.id.String())
			time.Sleep(time.Second)
		}
		// get next request
		req := c.Algorithm.Next(c.id, c.bf, c.t.bf)
		if req == nil {
			log.Debugf("No more pieces to request from %s", c.id.String())
			time.Sleep(time.Second)
			continue
		}
		// send request
		c.Send(req.ToWireMessage())
	}
	log.Debugf("peer %s is 'done'", c.id.String())
	// done downloading
	if c.Done != nil {
		c.Done()
	}
	log.Debugf("Close connection to %s", c.id.String())
	c.Close()
}
