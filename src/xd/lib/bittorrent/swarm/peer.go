package swarm

import (
	"io"
	"net"
	"sync"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/bittorrent/extensions"
	"xd/lib/common"
	"xd/lib/log"
)

// a peer connection
type PeerConn struct {
	inbound             bool
	closing             bool
	c                   net.Conn
	id                  common.PeerID
	t                   *Torrent
	send                chan *common.WireMessage
	bf                  *bittorrent.Bitfield
	peerChoke           bool
	peerInterested      bool
	usChoke             bool
	usInterseted        bool
	Done                func()
	keepalive           *time.Ticker
	lastSend            time.Time
	tx                  float32
	lastRecv            time.Time
	rx                  float32
	downloading         []*common.PieceRequest
	ourOpts             *extensions.ExtendedOptions
	theirOpts           *extensions.ExtendedOptions
	MaxParalellRequests int
	access              sync.Mutex
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
	p.MaxParalellRequests = 3
	p.send = make(chan *common.WireMessage, 8)
	p.keepalive = time.NewTicker(time.Minute)
	p.downloading = []*common.PieceRequest{}
	// TODO: hard coded
	return p
}

func (c *PeerConn) start() {
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

func (c *PeerConn) gotDownload(p *common.PieceData) {
	c.access.Lock()
	var downloading []*common.PieceRequest
	for _, r := range c.downloading {
		if r.Matches(p) {
			c.t.pt.handlePieceData(p)
		} else {
			downloading = append(downloading, r)
		}
	}
	c.downloading = downloading
	c.access.Unlock()
}

func (c *PeerConn) cancelDownload(req *common.PieceRequest) {
	c.access.Lock()
	var downloading []*common.PieceRequest
	for _, r := range c.downloading {
		if r.Equals(req) {
			c.t.pt.canceledRequest(r)
		} else {
			downloading = append(downloading, r)
		}
	}
	c.downloading = downloading
	c.access.Unlock()
}

func (c *PeerConn) numDownloading() int {
	c.access.Lock()
	i := len(c.downloading)
	c.access.Unlock()
	return i
}

func (c *PeerConn) queueDownload(req *common.PieceRequest) {
	if c.closing {
		c.clearDownloading()
		return
	}
	c.access.Lock()
	c.downloading = append(c.downloading, req)
	log.Debugf("ask %s for %d %d %d", c.id.String(), req.Index, req.Begin, req.Length)
	c.Send(req.ToWireMessage())
	c.access.Unlock()
}

func (c *PeerConn) clearDownloading() {
	c.access.Lock()
	for _, r := range c.downloading {
		c.t.pt.canceledRequest(r)
	}
	c.downloading = []*common.PieceRequest{}
	c.access.Unlock()
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
	for _, r := range c.downloading {
		c.t.pt.canceledRequest(r)
	}
	c.downloading = nil
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
		msg, e := c.Recv()
		err = e
		if err == nil {
			if msg.KeepAlive() {
				log.Debugf("keepalive from %s", c.id)
				continue
			}
			msgid := msg.MessageID()
			log.Debugf("%s from %s", msgid.String(), c.id.String())
			if msgid == common.BitField {
				isnew := false
				if c.bf == nil {
					isnew = true
				}
				c.bf = bittorrent.NewBitfield(c.t.MetaInfo().Info.NumPieces(), msg.Payload())
				log.Debugf("got bitfield from %s", c.id.String())
				// TODO: determine if we are really interested
				m := common.NewInterested()
				c.Send(m)
				if isnew {
					go c.runDownload()
				}
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
					c.gotDownload(d)
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
	if c.lastSend.Before(tm) {
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
					r := msg.GetPieceRequest()
					c.cancelDownload(r)
				} else {
					log.Debugf("writing %d bytes", msg.Len())
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
	pendingTry := 0
	for !c.t.Done() && c.send != nil {
		if c.RemoteChoking() {
			time.Sleep(time.Second)
			continue
		}
		// pending request
		p := c.numDownloading()
		if p >= c.MaxParalellRequests {
			log.Debugf("too many pending requests %d, waiting", p)
			if pendingTry > 5 {
				c.Close()
				return
			}
			pendingTry++
			time.Sleep(time.Second * 10)
			continue
		}
		pendingTry = 0
		r := c.t.pt.nextRequestForDownload(c.bf)
		if r == nil {
			log.Debugf("no next piece to download for %s", c.id.String())
			time.Sleep(time.Second)
		} else {
			c.queueDownload(r)
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
