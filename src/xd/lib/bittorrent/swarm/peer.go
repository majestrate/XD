package swarm

import (
	"io"
	"net"
	"time"
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
	// done callback
	Done func()
	// keepalive ticker
	keepalive *time.Ticker
	// current request
	req *bittorrent.PieceRequest
	// current piece
	piece *cachedPiece
}

func makePeerConn(c net.Conn, t *Torrent, id common.PeerID) *PeerConn {
	p := new(PeerConn)
	p.c = c
	p.t = t
	p.peerChoke = true
	p.usChoke = true
	copy(p.id[:], id[:])
	p.send = make(chan *bittorrent.WireMessage)
	p.keepalive = time.NewTicker(time.Minute)
	return p
}

func (c *PeerConn) start() {
	go c.runDownload()
	go c.runReader()
	go c.runWriter()
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
	log.Debugf("got %d bytes from %s", msg.Len(), c.id)
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
	c.keepalive.Stop()
	log.Debugf("%s closing connection", c.id.String())
	if c.send != nil {
		chnl := c.send
		c.send = nil
		time.Sleep(time.Second / 10)
		close(chnl)
		if c.piece != nil {
			c.t.cancelPiece(uint32(c.piece.piece.Index))
		}
	}
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
				var m *bittorrent.WireMessage
				// TODO: determine if we are really interested
				m = bittorrent.NewWireMessage(bittorrent.Interested, nil)
				c.Send(m)
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
				d := msg.GetPieceData()
				if c.req == nil {
					// no pending request
					log.Warnf("got unwarranted piece data from %s", c.id.String())
				} else if d.Index == c.req.Index && d.Begin == c.req.Begin {
					c.piece.put(int(d.Begin), d.Data)
					if c.piece.done() {
						c.t.storePiece(c.piece.piece)
						c.piece = nil
						c.req = nil
					} else {
						c.req = c.nextBlock()
						if c.req != nil {
							c.Send(c.req.ToWireMessage())
						} else {
							c.t.cancelPiece(uint32(c.piece.piece.Index))
							c.piece = nil
						}
					}
				} else {
					log.Warnf("got undesired piece data from %s", c.id.String())
				}
				continue
			}
			if msgid == bittorrent.Have {

				continue
			}
		}
	}
	if err != io.EOF {
		log.Errorf("%s read error: %s", c.id, err)
	}
	c.Close()
}

func (c *PeerConn) busy() bool {
	return c.piece != nil
}

func (c *PeerConn) getPiece(idx uint32) {
	log.Debugf("%s get piece %d", c.id.String(), idx)
	c.t.markPieceInProgress(idx, c)
	sz := c.t.MetaInfo().Info.PieceLength
	c.piece = new(cachedPiece)
	c.piece.piece = &common.Piece{
		Index: int64(idx),
		Data:  make([]byte, sz),
	}
	c.piece.progress = make([]byte, sz)
	c.req = c.nextBlock()
	c.Send(c.req.ToWireMessage())
}
func (c *PeerConn) nextBlock() *bittorrent.PieceRequest {
	off := c.piece.nextOffset()
	if off == -1 {
		return nil
	}
	b := uint32(off)
	return &bittorrent.PieceRequest{
		Index:  uint32(c.piece.piece.Index),
		Begin:  b,
		Length: BlockSize,
	}
}

func (c *PeerConn) sendKeepAlive() error {
	log.Debugf("send keepalive to %s", c.id.String())
	return bittorrent.KeepAlive().Send(c.c)
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
				if c.RemoteChoking() && msg.MessageID() == bittorrent.Request {
					// drop
					log.Debugf("drop request because choke")
				} else {
					log.Debugf("write message %s %d bytes", msg.MessageID(), msg.Len())
					err = msg.Send(c.c)
				}
			}
		}
	}
	log.Errorf("write loop ended: %s", err)
	c.Close()
}

// run download loop
func (c *PeerConn) runDownload() {
	for !c.t.Done() && c.send != nil {
		// check for choke
		if c.t.Choke(c.id) {
			c.Choke()
			continue
		} else {
			c.Unchoke()
		}

		if c.RemoteChoking() {
			time.Sleep(time.Second)
			continue
		}
		if c.req != nil {
			time.Sleep(time.Second)
			continue
		}
		if c.busy() {
			time.Sleep(time.Second)
			continue
		}
		remote := c.bf
		if remote == nil {
			log.Debugf("%s has not bitfield", c.id.String())
			time.Sleep(time.Second)
			continue
		}
		local := c.t.Bitfield()
		set := 0
		for remote.Has(set) {
			if local.Has(set) || c.t.pieceRequested(uint32(set)) {
				set++
			} else {
				break
			}
		}
		if set < local.Length {
			// this blocks
			c.getPiece(uint32(set))
		} else {
			// wut
			log.Debugf("%s could not request %d", c.id.String(), set)
		}
	}
	log.Debugf("peer %s is 'done'", c.id.String())
	// done downloading
	if c.Done != nil {
		c.Done()
	}
	log.Debugf("Close connection to %s", c.id.String())
	c.Close()
}
