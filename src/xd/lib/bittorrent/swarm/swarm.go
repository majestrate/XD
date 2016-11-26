package swarm

import (
	"net"
	"os"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/metainfo"
	"xd/lib/network"
	"xd/lib/storage"
	"xd/lib/tracker"
)

type Swarm struct {
	net network.Network
	Torrents *Holder
	id common.PeerID
}

// wait until we get a network context
func (sw *Swarm) WaitForNetwork() {
	for sw.net == nil {
		time.Sleep(time.Second)
	}
}

// add new torrent
func (sw *Swarm) AddTorrent(meta_fname string) (err error) {
	info := new(metainfo.TorrentFile)
	var f *os.File
	f, err = os.Open(meta_fname)
	if err == nil {
		err = info.BDecode(f)
		f.Close()
		if err == nil {
			var t storage.Torrent
			t, err = sw.Torrents.st.OpenTorrent(info)
			if err == nil {
				name := t.MetaInfo().TorrentName()
				log.Debugf("allocate space for %s", name)
				err = t.Allocate()
				if err != nil {
					return
				}
				log.Debugf("verify all pieces for %s", name)
				err = t.VerifyAll()
				if err != nil {
					return
				}
				sw.WaitForNetwork()
				sw.Torrents.addTorrent(t)
				tr := sw.Torrents.GetTorrent(t.Infohash())
				if tr != nil {
					sw.startTorrent(tr)
				}
			}
		}
	}
	return
}

func (sw *Swarm) startTorrent(t *Torrent) {
	// give network
	t.Net = sw.net
	// give peerid
	t.id = sw.id
	// add trackers
	info := t.MetaInfo()
	for _, u := range info.GetAllAnnounceURLS() {
		tr := tracker.FromURL(sw.net, u)
		if tr != nil {
			t.Trackers = append(t.Trackers, tr)
		}
	}
	// start annoucing
	go t.StartAnnouncing()
}

func (sw *Swarm) Run() (err error) {
	sw.WaitForNetwork()
	log.Infof("swarm obtained network address: %s", sw.net.Addr())

	// set up announcers

	sw.Torrents.ForEachTorrent(sw.startTorrent)
	
	for err == nil {
		var c net.Conn
		c, err = sw.net.Accept()
		if err == nil {
			log.Debugf("got inbound bittorrent connection from %s", c.RemoteAddr())
			go sw.inboundConn(c)
		}
	}
	return
}


// got inbound connection
func (sw *Swarm) inboundConn(c net.Conn) {
	h := new(bittorrent.Handshake)
	log.Debug("read bittorrent handshake")
	err := h.Recv(c)
	if err != nil {
		log.Warn("read bittorrent handshake failed, closing connection")
		// read error
		c.Close()
		return
	}
	t := sw.Torrents.GetTorrent(h.Infohash)
	if t == nil {
		log.Warnf("we don't have torrent with infohash %s, closing connection", h.Infohash.Hex())
		// no such torrent
		c.Close()
		return
	}

	// make peer conn
	p := makePeerConn(c, t, h.PeerID)
	
	// reply to handshake
	copy(h.PeerID[:], sw.id[:])
	err = h.Send(c)
	if err != nil {
		log.Warnf("didn't send bittorrent handshake reply: %s, closing connection", err)
		// write error
		c.Close()
		return
	}
	go p.runWriter()
	go p.runReader()
	t.OnNewPeer(p)
}

func (sw *Swarm) AddTorrents() (err error) {
	var ts []storage.Torrent
	ts, err = sw.Torrents.st.OpenAllTorrents()
	if err == nil {
		for _, t := range ts {
			name := t.MetaInfo().TorrentName()
			log.Debugf("allocate space for %s", name)
			err = t.Allocate()
			if err != nil {
				break
			}
			log.Debugf("verify all pieces for %s", name)
			err = t.VerifyAll()
			if err != nil {
				break
			}
			sw.Torrents.addTorrent(t)
			log.Infof("added torrent %s", name)
		}
	}
	return
}

func (sw *Swarm) SetNetwork(net network.Network) {
	sw.net = net
}

func NewSwarm(storage storage.Storage) *Swarm {
	sw := &Swarm{
		Torrents: &Holder{
			st: storage,
			torrents: make(map[common.Infohash]*Torrent),
		},
	}
	sw.id = common.GeneratePeerID()
	log.Infof("generated peer id %s", sw.id.String())
	return sw
}
