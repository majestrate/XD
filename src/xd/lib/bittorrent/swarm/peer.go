package swarm

import (
	"io"
	"net"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/bittorrent/extensions"
	"xd/lib/common"
	"xd/lib/log"
)

// a peer connection
type PeerConn struct {
	inbound        bool
	closing        bool
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
	ourOpts        *extensions.ExtendedOptions
	theirOpts      *extensions.ExtendedOptions
}

// get stats for this connection
func (c *PeerConn) Stats() (st *PeerConnStats) {
	st = new(PeerConnStats)
	st.TX = c.tx
	st.RX = c.rx
	st.Addr = c.c.RemoteAddr().String()
	st.ID = c.id.String()
	return
}

func makePeerConn(c net.Conn, t *Torrent, id common.PeerID, ourOpts *extensions.ExtendedOptions) *PeerConn {
	p := new(PeerConn)
	p.c = c
	p.t = t
	p.ourOpts = ourOpts
	p.peerChoke = true
	p.usChoke = true
	copy(p.id[:], id[:])
	p.send = make(chan *common.WireMessage)
	p.keepalive = time.NewTicker(time.Minute)
	return p
}

func (c *PeerConn) start() {
	go c.runDownload()
	go c.runReader()
	go c.runWriter()
}

// queue a send of a bittorrent wire message to this peer
func (c *PeerConn) Send(msg *common.WireMessage) {
	if !c.closing {
		c.send <- msg
	}
}

// recv a bittorrent wire message (blocking)
func (c *PeerConn) Recv() (msg *common.WireMessage, err error) {
	// hack
	msg = common.KeepAlive()
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
	if c.closing {
		return
	}
	c.closing = true
	c.t.pt.canceledRequest(c.r)
	c.keepalive.Stop()
	log.Debugf("%s closing connection", c.id.String())
	if c.send != nil {
		chnl := c.send
		c.send = nil
		close(chnl)
	}
	c.c.Close()
	if c.inbound {
		c.t.removeIBConn(c)
	} else {
		c.t.removeOBConn(c)
	}
}

// run read loop
func (c *PeerConn) runReader() {
	var err error
	for err == nil {
		msg, err := c.Recv()
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
				// TODO: determine if we are really interested
				m := common.NewInterested()
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
				if d == nil {
					log.Warnf("invalid piece data message from %s", c.id.String())
					c.Close()
				} else {
					if c.r != nil && c.r.Index == d.Index && c.r.Begin == d.Begin && c.r.Length == uint32(len(d.Data)) {
						c.t.pt.handlePieceData(d)
					} else {
						log.Warnf("unwarrented piece data from %s", c.id.String())
						c.Close()
					}
					c.r = nil
				}
				continue
			}
			if msgid == common.Have && c.bf != nil {
				// update bitfield
				idx := msg.GetHave()
				c.bf.Set(idx)
				if c.t.Bitfield().Has(idx) {
					// not interested
					c.Send(common.NewNotInterested())
				} else {
					c.Send(common.NewInterested())
				}
				continue
			}
			if msgid == common.Cancel {
				// TODO: check validity
				r := msg.GetPieceRequest()
				c.t.pt.canceledRequest(r)
				continue
			}
			if msgid == common.Extended {
				// handle extended options
				opts := extensions.FromWireMessage(msg)
				if opts == nil {
					log.Warnf("failed to parse extended options for %s", c.id.String())
				} else {
					c.handleExtendedOpts(opts)
				}
			}
		}
	}
	if err != io.EOF {
		log.Errorf("%s read error: %s", c.id, err)
	}
	c.Close()
}

func (c *PeerConn) handleExtendedOpts(opts *extensions.ExtendedOptions) {
	log.Debugf("got extended opts from '%s'", opts.Version)
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
	for err == nil && !c.closing {
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
					c.r.Length = 0
				} else {
					err = msg.Send(c.c)
					log.Debugf("wrote message %s %d bytes", msg.MessageID(), msg.Len())
				}
			} else {
				break
			}
		}
	}
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
		if c.r.Length == 0 {
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
		return
	}
	log.Debugf("peer %s is 'done'", c.id.String())

	// done downloading
	if c.Done != nil {
		c.Done()
	}
}
