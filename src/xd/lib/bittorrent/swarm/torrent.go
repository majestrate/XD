package swarm

import (
	"bytes"
	"net"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/metainfo"
	"xd/lib/network"
	"xd/lib/storage"
	"xd/lib/tracker"
)

// an event triggered when we get an inbound wire message from a peer we are connected with on this torrent
type wireEvent struct {
	c *PeerConn
	msg *bittorrent.WireMessage
}

type Torrent struct {
	// network context
	Net network.Network
	Trackers []tracker.Announcer
	announcer *time.Ticker
	// our peer id
	id common.PeerID
	st storage.Torrent
	bf *bittorrent.Bitfield
	recv chan wireEvent
}

// start annoucing on all trackers
func (t *Torrent) StartAnnouncing() {
	for _, tr := range t.Trackers {
		t.Announce(tr, "started")
	}
	if t.announcer == nil {
		t.announcer = time.NewTicker(time.Second)
	}
	go t.pollAnnounce()
}

// stop annoucing on all trackers
func (t *Torrent) StopAnnouncing() {
	if t.announcer != nil {
		t.announcer.Stop()
	}
	for _, tr := range t.Trackers {
		t.Announce(tr, "stopped")
	}
}

// poll announce ticker channel and issue announces
func (t *Torrent) pollAnnounce() {
	for {
		_, ok := <- t.announcer.C
		if ! ok {
			// done
			return
		}
		for _, tr := range t.Trackers {
			if tr.ShouldAnnounce() {
				go t.Announce(tr, "")
			}
		}
	}
}

// do an announce
func (t *Torrent) Announce(tr tracker.Announcer, event string) {
	req := &tracker.Request{
		Infohash: t.st.Infohash(),
		PeerID: t.id,
		IP: t.Net.Addr(),
		Port: 6881,
		Event: event,
		NumWant: 10, // TODO: don't hardcode
	}
	resp, err := tr.Announce(req)
	if err == nil {
		for _, p := range resp.Peers {
			a, e := p.Resolve(t.Net)
			if e == nil {
				// no error resolving
				go t.AddPeer(a, p.ID)
			} else {
				log.Warnf("failed to resolve peer %s", e.Error())
			}
		}
	} else {
		log.Warnf("failed to announce to %s: %s", tr.Name(), err)
	}
}

// connect to a new peer for this swarm, blocks
func (t *Torrent) AddPeer(a net.Addr, id common.PeerID) {
	c, err := t.Net.Dial(a.Network(), a.String())
	if err == nil {
		// connected
		ih := t.st.Infohash()
		// build handshake
		h := new(bittorrent.Handshake)
		copy(h.Infohash[:], ih[:])
		copy(h.PeerID[:], t.id[:])
		// send handshake
		err = h.Send(c)
		if err == nil {
			// get response to handshake
			err = h.Recv(c)
			if err == nil {
				if bytes.Equal(ih[:], h.Infohash[:]) {
					// infohashes match
					pc := makePeerConn(c, t, h.PeerID)
					t.OnNewPeer(pc)
					return
				}
			}
		}
		log.Warnf("didn't complete handshake with peer: %s", err)
		// bad thing happened
		c.Close()
		return
	}
	log.Infof("didn't connect to %s: %s", a, err)
}

func (t *Torrent) MetaInfo() *metainfo.TorrentFile {
	return t.st.MetaInfo()
}

func (t *Torrent) OnNewPeer(c *PeerConn) {
	log.Infof("New peer (%s) for %s", c.id.String(), t.st.Infohash().Hex())
	// send our bitfields to them
	c.Send(t.bf.ToWireMessage())
}

func (t *Torrent) OnWireMessage(c *PeerConn, msg *bittorrent.WireMessage) {
	t.recv <- wireEvent{c, msg}
}

func (t *Torrent) Run() {
	for {
		ev, ok := <- t.recv
		if !ok {
			// channel closed
			return
		}
		if ev.msg.KeepAlive() {
			log.Debugf("got keepalive from %s", ev.c.id)
			continue
		}
		id := ev.msg.MessageID()
		log.Debugf("peer %s got message %d", ev.c.id, id)
	}
}
