package swarm

import (
	"net"
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

// an event triggered when we want to connect to a remote peer
type connectEvent struct {
	addr net.Addr
	id common.PeerID
}

type Torrent struct {
	// network context
	Net network.Network
	Trackers []tracker.Announcer
	st storage.Torrent
	bf *bittorrent.Bitfield
	recv chan wireEvent
	connect chan connectEvent
}

// start annoucing on all trackers
func (t *Torrent) StartAnnouncing() {
	
}

// stop annoucing on all trackers
func (t *Torrent) StopAnnouncing() {

}

func (t *Torrent) Announce(tr tracker.Announcer) (err error) {
	req := &tracker.Request{

	}
	var resp *tracker.Response
	resp, err = tr.Announce(req)
	if err == nil {
		for _, p := range resp.Peers {
			a, e := p.Resolve(t.Net)
			if e == nil {
				// no error resolving
				t.AddPeer(a, p.ID)
			} else {
				log.Warnf("failed to resolve peer %s", e.Error())
			}
		}
	}
	return
}

// connect to a new peer for this swarm
func (t *Torrent) AddPeer(a net.Addr, id common.PeerID) {
	t.connect <- connectEvent{a, id}
}

func (t *Torrent) MetaInfo() *metainfo.TorrentFile {
	return t.st.MetaInfo()
}

func (t *Torrent) OnNewPeer(c *PeerConn) {
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
			continue
		}
		id := ev.msg.MessageID()
		if id == 5 {
			// we got a bitfield
		}	
	}
}
