package swarm

import (
	"net"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/bittorrent/extensions"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/sync"
	"xd/lib/util"
)

const DefaultMaxParallelRequests = 4

// a peer connection
type PeerConn struct {
	readBuff            [common.MaxWireMessageSize + 4]byte
	sendPieceBuff       [BlockSize]byte
	inbound             bool
	c                   net.Conn
	id                  common.PeerID
	t                   *Torrent
	send                chan common.WireMessage
	bf                  *bittorrent.Bitfield
	peerChoke           bool
	peerInterested      bool
	usChoke             bool
	usInterested        bool
	Done                func()
	lastSend            time.Time
	tx                  *util.Rate
	lastRecv            time.Time
	rx                  *util.Rate
	downloading         []common.PieceRequest
	ourOpts             extensions.Message
	theirOpts           extensions.Message
	MaxParalellRequests int
	access              sync.Mutex
	close               chan bool
	statsTicker         *time.Ticker
	closing             bool
	uploading           bool
}

func (c *PeerConn) Bitfield() *bittorrent.Bitfield {
	if c.bf != nil {
		return c.bf.Copy()
	}
	return nil
}

// get stats for this connection
func (c *PeerConn) Stats() (st *PeerConnStats) {
	st = &PeerConnStats{}
	st.TX = c.tx.Mean()
	st.RX = c.rx.Mean()
	st.Addr = c.c.RemoteAddr().String()
	st.ID = c.id.String()
	st.UsInterested = c.usInterested
	st.ThemInterested = c.peerInterested
	st.UsChoking = c.usChoke
	st.ThemChoking = c.peerChoke
	st.Client = util.ClientNameFromID(c.id[:])
	st.Downloading = c.numDownloading() > 0
	st.Inbound = c.inbound
	st.Uploading = c.uploading
	if c.bf != nil {
		st.Bitfield.CopyFrom(c.bf)
	}
	return
}

func makePeerConn(c net.Conn, t *Torrent, id common.PeerID, ourOpts extensions.Message) *PeerConn {
	p := t.getNextPeer()
	p.c = c
	p.t = t
	p.tx = util.NewRate(10)
	p.rx = util.NewRate(10)
	p.statsTicker = time.NewTicker(time.Second)
	p.ourOpts = ourOpts
	p.peerChoke = true
	p.usChoke = true
	p.usInterested = true
	copy(p.id[:], id[:])
	p.MaxParalellRequests = t.MaxRequests
	p.downloading = []common.PieceRequest{}
	p.send = make(chan common.WireMessage, 128)
	return p
}

func (c *PeerConn) run() {
	for {
		select {
		case <-c.statsTicker.C:
			c.tx.Tick()
			c.rx.Tick()
		case <-c.close:
			c.doClose()
			return
		case msg := <-c.send:
			c.doSend(msg)
		}
	}
}

func (c *PeerConn) start() {
	go c.run()
	go c.runReader()
}

func (c *PeerConn) doSend(msg common.WireMessage) {
	if msg != nil {
		now := time.Now()
		c.lastSend = now
		if c.RemoteChoking() && msg.MessageID() == common.Request {
			// drop
			log.Debugf("cancel request because choke")
			c.cancelDownload(msg.GetPieceRequest())
			return
		}
		log.Debugf("writing %d bytes", msg.Len())
		err := util.WriteFull(c.c, msg)
		if err == nil {
			if msg.MessageID() == common.Piece {
				n := uint64(msg.Len())
				c.tx.AddSample(n)
				c.t.statsTracker.AddSample(RateUpload, n)
			}
		} else {
			log.Debugf("write error: %s", err.Error())
			c.doClose()
		}
	}
}

// queue a send of a bittorrent wire message to this peer
func (c *PeerConn) Send(msg common.WireMessage) {
	if c.send != nil {
		c.send <- msg
	}
}

func (c *PeerConn) recv(msg common.WireMessage) (err error) {
	c.lastRecv = time.Now()
	if (!msg.KeepAlive()) && msg.MessageID() == common.Piece {
		n := uint64(msg.Len())
		c.rx.AddSample(n)
		c.t.statsTracker.AddSample(RateDownload, n)
	}
	log.Debugf("got %d bytes from %s", msg.Len(), c.id)
	err = c.inboundMessage(msg)
	return
}

// send choke
func (c *PeerConn) Choke() {
	if c.usChoke {
		log.Warnf("multiple chokes sent to %s", c.id.String())
	} else {
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

func (c *PeerConn) gotDownload(p common.PieceData) {
	c.access.Lock()
	var downloading []common.PieceRequest
	for idx := range c.downloading {
		if c.downloading[idx].Matches(&p) {
			c.t.pt.handlePieceData(p)
		} else {
			downloading = append(downloading, c.downloading[idx])
		}
	}
	c.downloading = downloading
	c.access.Unlock()
}

func (c *PeerConn) cancelDownload(req common.PieceRequest) {
	c.access.Lock()
	var downloading []common.PieceRequest
	for _, r := range c.downloading {
		if r.Equals(&req) {
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

func (c *PeerConn) queueDownload(req common.PieceRequest) {
	c.access.Lock()
	c.downloading = append(c.downloading, req)
	c.access.Unlock()
	log.Debugf("ask %s for %d %d %d", c.id.String(), req.Index, req.Begin, req.Length)
	c.Send(req.ToWireMessage())
}

func (c *PeerConn) clearDownloading() {
	c.access.Lock()
	for _, r := range c.downloading {
		c.Send(r.Cancel())
		c.t.pt.canceledRequest(r)
	}
	c.downloading = []common.PieceRequest{}
	c.access.Unlock()
}

// returns true if the remote peer has piece with given index
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

func (c *PeerConn) cancelPendingDownloads() {
	c.access.Lock()
	for _, r := range c.downloading {
		c.t.pt.canceledRequest(r)
		c.Send(r.Cancel())
	}
	c.downloading = []common.PieceRequest{}
	c.access.Unlock()
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
	c.close <- true
}

func (c *PeerConn) doClose() {
	c.send = nil
	for _, r := range c.downloading {
		c.t.pt.canceledRequest(r)
	}
	c.downloading = nil
	log.Debugf("%s closing connection", c.id.String())
	if c.inbound {
		c.t.removeIBConn(c)
	} else {
		c.t.removeOBConn(c)
	}
	c.c.Close()
}

// run read loop
func (c *PeerConn) runReader() {
	err := common.ReadWireMessages(c.c, c.recv, c.readBuff[:])
	if err != nil {
		log.Debugf("PeerConn() reader failed: %s", err.Error())
	}
	c.Close()
}

func (c *PeerConn) cancelPiece(idx uint32) {
	c.access.Lock()
	downloading := c.downloading
	c.downloading = []common.PieceRequest{}
	for _, r := range downloading {
		if r.Index == idx {
			c.Send(r.Cancel())
		} else {
			c.downloading = append(c.downloading, r)
		}
	}
	c.access.Unlock()
}

func (c *PeerConn) checkInterested() {
	bf := c.t.Bitfield()
	if c.bf.XOR(bf).CountSet() > 0 {
		c.usInterested = true
		m := common.NewInterested()
		c.Send(m)
	} else {
		c.usInterested = false
		m := common.NewNotInterested()
		c.Send(m)
	}
}

func (c *PeerConn) metaInfoDownload() {
	if !c.t.Ready() && c.theirOpts.MetaData() {
		if c.theirOpts.MetainfoSize != nil {
			l := *c.theirOpts.MetainfoSize
			if c.t.metaInfo == nil || len(c.t.metaInfo) == 0 {
				// set meta info
				c.t.metaInfo = make([]byte, l)
				l = 1 + (l / (16 * 1024))
				log.Debugf("bitfield is %d bits", l)
				c.t.pendingInfoBF = bittorrent.NewBitfield(l, nil)
				c.t.requestingInfoBF = bittorrent.NewBitfield(l, nil)
			} else {
				log.Debugf("metainfo len=%d", len(c.t.metaInfo))
			}
		} else {
			log.Debugf("no metainfo size: %q", c.theirOpts)
		}
		id, ok := c.theirOpts.Extensions[extensions.UTMetaData.String()]
		if ok {
			var md extensions.MetaData
			md.Type = extensions.UTRequest
			r := c.t.nextMetaInfoReq()
			if r != nil {
				md.Piece = *r
				m := &extensions.Message{ID: uint8(id), Raw: md.Bytes()}
				log.Debugf("asking for info piece %d", md.Piece)
				c.Send(m.ToWireMessage())
			} else {
				log.Debugf("no more pieces desired")
			}
		} else {
			log.Debug("ut_metadata not found?")
		}
	}
}

func (c *PeerConn) inboundMessage(msg common.WireMessage) (err error) {

	if msg.KeepAlive() {
		log.Debugf("keepalive from %s", c.id)
		return
	}
	msgid := msg.MessageID()
	log.Debugf("%s from %s", msgid.String(), c.id.String())
	if msgid == common.BitField {
		isnew := false
		if c.bf == nil {
			isnew = true
		}
		if c.t.Ready() {
			c.bf = bittorrent.NewBitfield(c.t.MetaInfo().Info.NumPieces(), msg.Payload())
			log.Debugf("got bitfield from %s", c.id.String())
			c.checkInterested()
			if isnew {
				c.Unchoke()
				c.Send(c.ourOpts.ToWireMessage())
			}
		} else {
			// empty bitfield
			bits := make([]byte, len(msg.Payload()))
			c.Send(common.NewWireMessage(common.BitField, bits))
			c.Send(c.ourOpts.ToWireMessage())
			c.metaInfoDownload()
		}
		if isnew {
			if c.t.Ready() {
				go c.runDownload()
			}
		}
		return
	}
	if msgid == common.Choke {
		c.remoteChoke()
		c.cancelPendingDownloads()
	}
	if msgid == common.UnChoke {
		c.remoteUnchoke()
	}
	if msgid == common.Interested {
		c.markInterested()
	}
	if msgid == common.NotInterested {
		c.markNotInterested()
	}
	if msgid == common.Request {
		c.uploading = true
		ev := msg.GetPieceRequest()
		c.t.handlePieceRequest(c, ev)
	}
	if msgid == common.Piece {
		msg.VisitPieceData(c.gotDownload)
	}

	if msgid == common.Have {
		// update bitfield
		idx := msg.GetHave()
		if c.bf != nil {
			c.bf.Set(idx)
			c.checkInterested()
		} else {
			// default to interested if we have no bitfield yet
			c.Send(common.NewInterested())
		}
	}
	if msgid == common.Cancel {
		// TODO: check validity
		//c.t.pt.canceledRequest(msg.GetPieceRequest())
	}
	if msgid == common.Extended {
		// handle extended options
		opts, err := extensions.FromWireMessage(msg)
		if err == nil {
			c.handleExtendedOpts(opts)
		} else {
			log.Warnf("failed to parse extended options for %s, %s", c.id.String(), err.Error())
		}
	}
	return
}

// handles an inbound pex message
func (c *PeerConn) handlePEX(msg extensions.Message) {
	var pex extensions.PEX
	err := msg.DecodePayload(&pex)
	if err == nil {
		c.handlePEXAdded(pex.Added)
	} else {
		log.Errorf("invalid pex message: %s", err.Error())
	}
}

// handle inbound PEX message payload
func (c *PeerConn) handlePEXAdded(msg string) {
	var peers []common.Peer
	l := len(msg) / 32
	for l > 0 {
		var p common.Peer
		// TODO: bounds check
		copy(p.Compact[:], msg[(l-1)*32:l*32])
		l--
		peers = append(peers, p)
	}
	c.t.addPeers(peers)
}

func (c *PeerConn) handlePEXAddedf(m interface{}) {
	// TODO: implement this
}

func (c *PeerConn) SupportsPEX() bool {
	return c.theirOpts.PEX()
}

func (c *PeerConn) sendPEX(connected, disconnected []byte) {
	id := c.theirOpts.Extensions[extensions.PeerExchange.String()]
	msg := extensions.NewPEX(uint8(id), connected, disconnected)
	c.Send(msg.ToWireMessage())
}

func (c *PeerConn) handleExtendedOpts(opts extensions.Message) {
	log.Debugf("got extended opts from %s: %q", c.id.String(), opts)
	if opts.ID == 0 {
		// handshake
		c.theirOpts = opts.Copy()
	} else {
		// lookup the extension number
		ext, ok := c.ourOpts.Lookup(opts.ID)
		if ok {
			if ext == extensions.PeerExchange.String() {
				// this is PEX message
				c.handlePEX(opts)
			} else if ext == extensions.XDHT.String() {
				// xdht message
				err := c.t.xdht.HandleMessage(opts, c.id)
				if err != nil {
					log.Warnf("error handling xdht message from %s: %s", c.id.String(), err.Error())
				}
			} else if ext == extensions.UTMetaData.String() {
				c.handleMetadata(opts)
			}
		} else {
			log.Warnf("peer %s gave us extension for message we do not have id=%d", c.id.String(), opts.ID)
		}

	}
}

func (c *PeerConn) askNextMetadata(id uint8) {
	r := c.t.nextMetaInfoReq()
	if r != nil {
		var m extensions.Message
		var msg extensions.MetaData
		msg.Type = extensions.UTRequest
		msg.Data = nil
		msg.Size = 0
		msg.Piece = *r
		m.ID = id
		m.Raw = msg.Bytes()
		log.Debugf("asking for info piece %d", msg.Piece)
		c.Send(m.ToWireMessage())
	} else {
		log.Debug("no more info pieces required")
	}
}

func (c *PeerConn) handleMetadata(m extensions.Message) {
	msg, err := extensions.ParseMetadata(m.Raw)
	if err == nil {
		if msg.Type == extensions.UTData {
			log.Debugf("got UTData: piece %d", msg.Piece)
			if !c.t.Ready() && msg.Size > 0 {
				c.t.putInfoSlice(msg.Piece, msg.Data)
				c.askNextMetadata(m.ID)
			}
		} else if msg.Type == extensions.UTReject {
			log.Debugf("ut_metadata rejected from %s", c.id.String())
			c.t.requestingInfoBF.Unset(msg.Piece)
		} else if msg.Type == extensions.UTRequest {
			if c.t.Ready() {
				idx := msg.Piece * (16 * 1024)
				pieces := c.t.metaInfo
				if uint32(len(pieces)) >= idx+(32*1024) {
					msg.Type = extensions.UTReject
				} else if uint32(len(pieces)) >= idx+(16*1024) {
					msg.Type = extensions.UTData
					msg.Data = pieces[idx:]
					msg.Size = uint32(len(msg.Data))
				} else {
					msg.Type = extensions.UTData
					msg.Data = pieces[idx : idx+(16*1024)]
					msg.Size = uint32(len(msg.Data))
				}
			} else {
				msg.Type = extensions.UTReject
			}
			m.Raw = msg.Bytes()
			c.Send(m.ToWireMessage())
		}
	} else {
		log.Errorf("failed to parse ut_metainfo message: %s", err.Error())
	}
}

func (c *PeerConn) sendKeepAlive() {
	tm := time.Now().Add(0 - (time.Minute * 2))
	if c.lastSend.Before(tm) {
		log.Debugf("send keepalive to %s", c.id.String())
		c.Send(common.KeepAlive)
	}
}

// run download loop
func (c *PeerConn) runDownload() {

	for !c.t.Done() && (c.usInterested || c.peerInterested) && !c.closing {
		if c.RemoteChoking() {
			log.Debugf("will not download this tick, %s is choking", c.id.String())
			time.Sleep(time.Second)
			continue
		}
		// pending request
		p := c.numDownloading()
		if p >= c.MaxParalellRequests {
			log.Debugf("max parallel reached for %s", c.id.String())
			time.Sleep(time.Second)
			continue
		}
		var r common.PieceRequest
		if c.t.pt.nextRequestForDownload(c.bf, &r) {
			c.queueDownload(r)
		} else {
			log.Debugf("no next piece to download for %s", c.id.String())
			time.Sleep(time.Second)
		}
	}
	// done downloading
	if c.Done != nil {
		c.Done()
	}
}
