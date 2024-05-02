package swarm

import (
	"github.com/majestrate/XD/lib/bittorrent"
	"github.com/majestrate/XD/lib/bittorrent/extensions"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/sync"
	"github.com/majestrate/XD/lib/util"
	"io"
	"net"
	"strconv"
	"time"
)

// a peer connection
type PeerConn struct {
	writeBuff           util.Buffer
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
	sentInterested      bool
	Done                func()
	lastSend            time.Time
	tx                  *util.Rate
	lastRecv            time.Time
	rx                  *util.Rate
	downloading         []*common.PieceRequest
	lastRequest         *common.PieceRequest
	ourOpts             extensions.Message
	theirOpts           extensions.Message
	MaxParalellRequests int
	access              sync.Mutex
	close               chan bool
	ticker              *time.Ticker
	tickstats           bool
	closing             bool
	uploading           bool
	runDownload         bool
	nextPieceRequest    time.Time
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
	p.ticker = time.NewTicker(time.Millisecond * 500)
	p.ourOpts = ourOpts
	p.peerChoke = true
	p.usChoke = true
	p.usInterested = true
	copy(p.id[:], id[:])
	p.MaxParalellRequests = t.MaxRequests
	p.downloading = []*common.PieceRequest{}
	p.send = make(chan common.WireMessage, 128)
	return p
}

func (c *PeerConn) appendSend(msg common.WireMessage) {
	if c.writeBuff.Len() > 1000 {
		if c.flushSend() != nil {
			c.closing = true
			c.doClose()
			return
		}
	}
	c.processWrite(&c.writeBuff, msg)
}

func (c *PeerConn) run() {
	for {
		select {
		case <-c.ticker.C:
			if c.flushSend() != nil {
				c.closing = true
				c.doClose()
				continue
			}
			if c.tickstats {
				c.tx.Tick()
				c.rx.Tick()
			}
			c.tickstats = !c.tickstats
		case <-c.close:
			c.doClose()
			return
		case msg := <-c.send:
			if msg == nil {
				continue
			}
			if msg.Len() > 1000 {
				if c.flushSend() == nil {
					// write big messages right away
					if c.processWrite(c.c, msg) != nil {
						c.closing = true
						c.doClose()
						continue
					}
				} else {
					c.closing = true
					c.doClose()
					continue
				}
			} else {
				c.appendSend(msg)
			}
		}
	}
}

func (c *PeerConn) start() {
	go c.run()
	go c.runReader()
}

func (c *PeerConn) flushSend() error {
	_, err := io.Copy(c.c, &c.writeBuff)
	c.writeBuff.Reset()
	return err
}

func (c *PeerConn) btPeer() (p common.Peer) {
	h, prt, _ := net.SplitHostPort(c.c.RemoteAddr().String())
	copy(p.ID[:], c.id[:])
	p.IP = h
	p.Port, _ = strconv.Atoi(prt)
	return
}

func (c *PeerConn) processWrite(w io.Writer, msg common.WireMessage) (err error) {
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
		err = util.WriteFull(w, msg)
		if err == nil {
			if msg.MessageID() == common.Piece {
				n := uint64(msg.Len())
				c.tx.AddSample(n)
				c.t.statsTracker.AddSample(RateUpload, n)
			}
		}
	}
	return
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

func (c *PeerConn) gotDownload(p *common.PieceData) {
	c.access.Lock()
	var downloading []*common.PieceRequest
	for idx := range c.downloading {
		if c.downloading[idx].Matches(p) {
			c.t.pt.handlePieceData(p)
		} else {
			downloading = append(downloading, c.downloading[idx])
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
	c.lastRequest = req
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
	c.downloading = []*common.PieceRequest{}
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
	c.downloading = []*common.PieceRequest{}
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
	c.ticker.Stop()
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
	c.downloading = []*common.PieceRequest{}
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
	if bf != nil && c.bf != nil && c.bf.XOR(bf).CountSet() > 0 {
		c.usInterested = true
		m := common.NewInterested()
		c.Send(m)
		c.sentInterested = true
	} else {
		c.usInterested = false
		m := common.NewNotInterested()
		c.sentInterested = true
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
		}
		id, ok := c.theirOpts.Extensions[extensions.UTMetaData.String()]
		if ok {
			var md extensions.MetaData
			md.Type = extensions.UTRequest
			r := c.t.nextMetaInfoReq()
			if r != nil {
				md.Piece = *r
				m := &extensions.Message{ID: uint8(id), PayloadRaw: md.Bytes()}
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
				c.runDownload = true
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
		if !c.sentInterested {
			c.checkInterested()
			c.Unchoke()
		}
	}
	if msgid == common.NotInterested {
		c.markNotInterested()
		if !c.sentInterested {
			c.checkInterested()
		}
	}
	if msgid == common.Request {
		c.uploading = true
		ev := msg.GetPieceRequest()
		if ev != nil {
			c.t.handlePieceRequest(c, ev)
		}
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
			c.Send(common.NewNotInterested())
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

func (c *PeerConn) handleLNPEX(m interface{}) {
	var peers []common.Peer
	pex, ok := m.(map[string]interface{})
	if ok {
		added, ok := pex["addedln"]
		if ok {
			l, ok := added.([]interface{})
			if ok {
				for idx := range l {
					p, ok := l[idx].(map[string]interface{})
					if ok {
						var peer common.Peer
						v, ok := p["ip"]
						if !ok {
							continue
						}
						peer.IP, ok = v.(string)
						if !ok {
							continue
						}
						v, ok = p["port"]
						if !ok {
							continue
						}
						port, ok := v.(int64)
						if !ok {
							continue
						}
						peer.Port = int(port)
						v, ok = p["peer id"]
						if !ok {
							continue
						}
						pid, ok := v.(string)
						if ok && len(pid) == 20 {
							copy(peer.ID[:], pid[:])
						} else {
							continue
						}
						peers = append(peers, peer)
					}
				}
			}
		}
		c.t.addPeers(peers)
	} else {
		log.Errorf("invalid pex message: %q", m)
	}
}

// handles an inbound pex message
func (c *PeerConn) handleI2PPEX(m interface{}) {

	pex, ok := m.(map[string]interface{})
	if ok {
		var added interface{}
		added, ok = pex["added"]
		if ok {
			c.handlePEXAdded(added)
		}
		added, ok = pex["added.f"]
		if ok {
			c.handlePEXAddedf(added)
		}
	} else {
		log.Errorf("invalid pex message: %q", m)
	}
}

// handle inbound PEX message payload
func (c *PeerConn) handlePEXAdded(m interface{}) {
	var peers []common.Peer
	msg := m.(string)
	l := len(msg) / 32
	log.Infof("%s got %d peers for %s", c.id.String(), l, c.t.st.Infohash().Hex())
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

func (c *PeerConn) SupportsI2PPEX() bool {
	return c.theirOpts.I2PPEX()
}

func (c *PeerConn) SupportsLNPEX() bool {
	return c.theirOpts.LNPEX()
}

func (c *PeerConn) sendI2PPEX(connected, disconnected []byte) {
	if len(connected) > 0 || len(disconnected) > 0 {
		id := c.theirOpts.Extensions[extensions.I2PPeerExchange.String()]
		msg := extensions.NewI2PPEX(uint8(id), connected, disconnected)
		c.Send(msg.ToWireMessage())
	}
}

func (c *PeerConn) sendLNPEX(connected, disconnected []common.Peer) {
	id := c.theirOpts.Extensions[extensions.LokinetPeerExchange.String()]
	msg := extensions.NewLNPEX(uint8(id), connected, disconnected)
	c.Send(msg.ToWireMessage())
}

func (c *PeerConn) handleExtendedOpts(opts extensions.Message) {
	if opts.ID == 0 {
		// handshake
		c.theirOpts = opts.Copy()
	} else {
		// lookup the extension number
		ext, ok := c.ourOpts.Lookup(opts.ID)
		if ok {
			if ext == extensions.I2PPeerExchange.String() {
				c.handleI2PPEX(opts.Payload)
			} else if ext == extensions.LokinetPeerExchange.String() {
				c.handleLNPEX(opts.Payload)
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
		m.PayloadRaw = msg.Bytes()
		log.Debugf("asking for info piece %d", msg.Piece)
		c.Send(m.ToWireMessage())
	} else {
		log.Debug("no more info pieces required")
	}
}

func (c *PeerConn) handleMetadata(m extensions.Message) {
	msg, err := extensions.ParseMetadata(m.PayloadRaw)
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
				pieces := c.t.getMetaInfo()
				if pieces == nil || len(pieces) == 0 {
					msg.Type = extensions.UTReject
				} else if uint32(len(pieces)) >= idx+(32*1024) {
					msg.Type = extensions.UTReject
				} else if uint32(len(pieces)) >= idx+(16*1024) {
					if idx < uint32(len(pieces)) {
						msg.Type = extensions.UTData
						msg.Data = pieces[idx:]
						msg.Size = uint32(len(msg.Data))
					} else {
						msg.Type = extensions.UTReject
					}
				} else {
					if idx+(16*1024) < uint32(len(pieces)) {
						msg.Type = extensions.UTData
						msg.Data = pieces[idx : idx+(16*1024)]
						msg.Size = uint32(len(msg.Data))
					} else {
						msg.Type = extensions.UTReject
					}
				}
			} else {
				msg.Type = extensions.UTReject
			}
			m.Payload = nil
			m.PayloadRaw = msg.Bytes()
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

// tick download stuff
func (c *PeerConn) tickDownload() {
	if !c.runDownload {
		return
	}
	if c.t.Done() {
		// done downloading
		if c.Done != nil {
			c.Done()
			c.Done = nil
		}
	} else if (c.usInterested || c.peerInterested) && !c.closing {
		if c.RemoteChoking() {
			//log.Debugf("will not download this tick, %s is choking", c.id.String())
			return
		}
		// pending request
		p := c.numDownloading()
		if p >= c.MaxParalellRequests {
			//log.Debugf("max parallel reached for %s", c.id.String())
			return
		}
		now := time.Now()
		if now.After(c.nextPieceRequest) {
			r := c.t.pt.NextRequest(c.bf, c.lastRequest)
			if r != nil {
				c.queueDownload(r)
			} else {
				c.nextPieceRequest = now.Add(time.Second / 4)
				log.Debugf("no next piece to download for %s", c.id.String())
			}
		}
	}
}
