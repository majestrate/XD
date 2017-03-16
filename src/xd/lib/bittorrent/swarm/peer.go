package swarm

import (
	"io"
	"net"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/log"
)

// connection statistics
type PeerConnStats struct {
	TX   float32
	RX   float32
	ID   common.PeerID
	Addr net.Addr
}

// a peer connection
type PeerConn struct {
	c              net.Conn
	id             common.PeerID
	t              *Torrent
	send           chan *common.WireMessage
	bf             *bittorrent.Bitfield
	peerChoke      bool
	peerInterested bool
	usChoke        bool
	usInterseted   bool
	Done           func()
	keepalive      *time.Ticker
	lastSend       time.Time
	tx             float32
	lastRecv       time.Time
	rx             float32
	r              *common.PieceRequest
}

// get stats for this connection
func (c *PeerConn) Stats() (st *PeerConnStats) {
	st = new(PeerConnStats)
	st.TX = c.tx
	st.RX = c.rx
	st.Addr = c.c.RemoteAddr()
	copy(st.ID[:], c.id[:])
	return
}

func makePeerConn(c net.Conn, t *Torrent, id common.PeerID) *PeerConn {
	p := new(PeerConn)
	p.c = c
	p.t = t
	p.peerChoke = true
	p.usChoke = true
	copy(p.id[:], id[:])
	p.send = make(chan *common.WireMessage, 8)
	p.keepalive = time.NewTicker(time.Minute)
	return p
}

func (c *PeerConn) start() {
	go c.runDownload()
	go c.runReader()
	go c.runWriter()
}

// send a bittorrent wire message to this peer
func (c *PeerConn) Send(msg *common.WireMessage) {
	if c.send == nil {
		log.Errorf("%s has no send channel but tried to send", c.id)
		return
	}
	c.send <- msg
}

// recv a bittorrent wire message (blocking)
func (c *PeerConn) Recv() (msg *common.WireMessage, err error) {
	msg = new(common.WireMessage)
	err = msg.Recv(c.c)
	log.Debugf("got %d bytes from %s", msg.Len(), c.id)
	now := time.Now()
	c.rx = float32(msg.Len()) / float32(now.Unix()-c.lastRecv.Unix())
	c.lastRecv = now
	return
}

// send choke
func (c *PeerConn) Choke() {
	if !c.usChoke {
		log.Debugf("choke peer %s", c.id.String())
		c.Send(common.NewWireMessage(common.Choke, nil))
		c.usChoke = true
	}
}

// send unchoke
func (c *PeerConn) Unchoke() {
	if c.usChoke {
		log.Debugf("unchoke peer %s", c.id.String())
		c.Send(common.NewWireMessage(common.UnChoke, nil))
		c.usChoke = false
	}
}

func (c *PeerConn) HasPiece(piece uint32) bool {
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
	addr := c.c.RemoteAddr()
	if c.r != nil {
		pc := c.t.pt.getPiece(c.r.Index)
		pc.cancel(c.r.Begin, c.r.Length)
	}
	c.keepalive.Stop()
	log.Debugf("%s closing connection", c.id.String())
	if c.send != nil {
		chnl := c.send
		c.send = nil
		time.Sleep(time.Second / 10)
		close(chnl)
	}
	c.c.Close()
	c.t.removePeer(addr)
}

// run read loop
func (c *PeerConn) runReader() {
	var err error
	for err == nil {
		var msg *common.WireMessage
		msg, err = c.Recv()
		if err == nil {
			if msg.KeepAlive() {
				log.Debugf("keepalive from %s", c.id)
				continue
			}
			msgid := msg.MessageID()
			log.Debugf("%s from %s", msgid.String(), c.id.String())
			if msgid == common.BitField {
				c.bf = bittorrent.NewBitfield(c.t.MetaInfo().Info.NumPieces(), msg.Payload())
				log.Debugf("got bitfield from %s", c.id.String())
				var m *common.WireMessage
				// TODO: determine if we are really interested
				m = common.NewWireMessage(common.Interested, nil)
				c.Send(m)
				continue
			}
			if msgid == common.Choke {
				c.remoteChoke()
				continue
			}
			if msgid == common.UnChoke {
				c.remoteUnchoke()
				continue
			}
			if msgid == common.Interested {
				c.markInterested()
				c.Unchoke()
				continue
			}
			if msgid == common.NotInterested {
				c.markNotInterested()
				continue
			}
			if msgid == common.Request {
				ev := msg.GetPieceRequest()
				c.t.onPieceRequest(c, ev)
				continue
			}
			if msgid == common.Piece {
				d := msg.GetPieceData()
				if c.r != nil && c.r.Index == d.Index && c.r.Begin == d.Begin && c.r.Length == uint32(len(d.Data)) {
					c.t.pt.handlePieceData(d)
				} else {
					log.Warnf("unwarrented piece data from %s", c.id.String())
					c.Close()
				}
				c.r = nil
				continue
			}
			if msgid == common.Have && c.bf != nil {
				// update bitfield
				c.bf.Set(msg.GetHave())
				if c.bf.AND(c.t.Bitfield()).CountSet() == 0 {
					// not interested
					c.Send(common.NewNotInterested())
				} else {
					c.Send(common.NewInterested())
				}
				continue
			}
		}
	}
	if err != io.EOF {
		log.Errorf("%s read error: %s", c.id, err)
	}
	c.Close()
}

func (c *PeerConn) sendKeepAlive() error {
	tm := time.Now().Add(0 - (time.Minute * 2))
	if c.lastSend.After(tm) {
		log.Debugf("send keepalive to %s", c.id.String())
		return common.KeepAlive().Send(c.c)
	}
	return nil
}

// run write loop
func (c *PeerConn) runWriter() {
	var err error
	for err == nil {
		select {
		case <-c.keepalive.C:
			err = c.sendKeepAlive()
		case msg, ok := <-c.send:
			if ok {
				now := time.Now()
				c.tx = float32(msg.Len()) / float32(now.Unix()-c.lastSend.Unix())
				c.lastSend = now
				if c.RemoteChoking() && msg.MessageID() == common.Request {
					// drop
					log.Debugf("drop request because choke")
					c.t.pt.canceledRequest(c.r)
					c.r = nil
				} else {
					log.Debugf("write message %s %d bytes", msg.MessageID(), msg.Len())
					err = msg.Send(c.c)
				}
			} else {
				break
			}
		}
	}
	log.Errorf("write loop ended: %s", err)
	c.Close()
}

// run download loop
func (c *PeerConn) runDownload() {
	for !c.t.Done() && c.send != nil {
		if c.RemoteChoking() {
			time.Sleep(time.Second)
			continue
		}
		// pending request
		if c.r != nil {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		c.r = c.t.pt.nextRequestForDownload(c.bf)
		if c.r == nil {
			log.Debugf("no next piece to download for %s", c.id.String())
			time.Sleep(time.Second)
		} else {
			log.Debugf("ask %s for %d %d %d", c.id.String(), c.r.Index, c.r.Begin, c.r.Length)
			c.Send(c.r.ToWireMessage())
		}
	}
	if c.send == nil {
		c.Close()
		log.Debugf("peer %s disconnected trying reconnect", c.id.String())
		go c.t.AddPeer(c.c.RemoteAddr(), c.id)
		return
	}
	log.Debugf("peer %s is 'done'", c.id.String())

	// done downloading
	if c.Done != nil {
		c.Done()
	}
	log.Debugf("Close connection to %s", c.id.String())
	c.Close()
}
