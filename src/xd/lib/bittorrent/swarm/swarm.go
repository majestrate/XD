package swarm

import (
	"bytes"
	"net"
	"net/http"
	"time"
	"xd/lib/bittorrent"
	"xd/lib/bittorrent/extensions"
	"xd/lib/common"
	"xd/lib/dht"
	"xd/lib/gnutella"
	"xd/lib/log"
	"xd/lib/metainfo"
	"xd/lib/network"
	"xd/lib/storage"
	"xd/lib/tracker"
	"xd/lib/util"
)

// a bittorrent swarm tracking many torrents
type Swarm struct {
	closing  bool
	net      network.Network
	Torrents Holder
	id       common.PeerID
	trackers map[string]tracker.Announcer
	xdht     dht.XDHT
	gnutella *gnutella.Swarm
}

func (sw *Swarm) Running() bool {
	return !sw.closing
}

// wait until we get a network context
func (sw *Swarm) WaitForNetwork() {
	for sw.net == nil {
		time.Sleep(time.Second)
	}
}

func (sw *Swarm) startTorrent(t *Torrent) {
	t.RemoveSelf = func() {
		sw.Torrents.removeTorrent(t.st.Infohash())
	}
	sw.WaitForNetwork()
	t.ObtainedNetwork(sw.net)
	t.xdht = &sw.xdht
	// give peerid
	t.id = sw.id
	// add open trackers
	for name := range sw.trackers {
		t.Trackers[name] = sw.trackers[name]
	}

	info := t.MetaInfo()
	for _, u := range info.GetAllAnnounceURLS() {
		tr := tracker.FromURL(u)
		if tr != nil {
			name := tr.Name()
			_, ok := t.Trackers[name]
			if !ok {
				t.Trackers[name] = tr
			}
		}
	}
	// handle messages
	t.Start()
}

// got inbound connection
func (sw *Swarm) inboundConn(c net.Conn) {
	var firstBytes [20]byte
	n, err := c.Read(firstBytes[:])
	if err != nil || n != 20 {
		log.Debug("failed to read first bytes")
		c.Close()
		return
	}
	if firstBytes[0] == 19 {
		// bittorrent
		var buff [68]byte
		copy(buff[:], firstBytes[:])
		n, err = c.Read(buff[20:])
		if err != nil || n != 48 {
			log.Debugf("failed to read bittorrent handshake: %d bytes", n)
			c.Close()
			return
		}
		h := new(bittorrent.Handshake)
		err := h.FromBytes(buff[:])
		if err != nil {
			log.Debug(err.Error())
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
		var opts *extensions.Message
		if h.Reserved.Has(bittorrent.Extension) {
			opts = extensions.New()
		}
		// reply to handshake
		var id common.PeerID
		copy(id[:], h.PeerID[:])
		copy(h.PeerID[:], sw.id[:])
		err = h.Send(c)
		if err != nil {
			log.Warnf("didn't send bittorrent handshake reply: %s, closing connection", err)
			// write error
			c.Close()
			return
		}
		// make peer conn
		p := makePeerConn(c, t, id, opts)
		t.onNewPeer(p)

	} else if bytes.Equal(firstBytes[:], []byte(gnutella.Handshake)) {
		// gnutella
		var delim [2]byte
		// discard crlf
		c.Read(delim[:])
		// do the rest of the handshake
		conn := gnutella.NewConn(c)
		err = conn.Handshake(sw.gnutella == nil)
		if err == nil && sw.gnutella != nil {
			log.Debug("got GNUTella Peer")
			sw.gnutella.AddInboundPeer(conn)
		} else {
			conn.Close()
		}
	} else {
		// unknown
		log.Debug("bad protocol handshake")
		c.Close()
		return
	}
}

// add a torrent to this swarm
func (sw *Swarm) AddTorrent(t storage.Torrent) (err error) {
	sw.Torrents.addTorrent(t)
	tr := sw.Torrents.GetTorrent(t.Infohash())
	go sw.startTorrent(tr)
	return
}

func (sw *Swarm) getCurrentBW() (bw SwarmBandwidth) {

	var rx, tx float64

	sw.Torrents.ForEachTorrent(func(t *Torrent) {
		p := t.GetStatus().Peers
		tx += p.TX()
		rx += p.RX()
	})

	bw.Upload = util.FormatRate(tx)
	bw.Download = util.FormatRate(rx)
	return
}

// run with network context
func (sw *Swarm) Run(n network.Network) (err error) {
	// give network to swarm
	sw.net = n
	// give network to torrents
	sw.Torrents.ForEachTorrent(func(t *Torrent) {
		t.ObtainedNetwork(n)
	})
	log.Debug("gave network context to torrents")
	// accept inbound connections
	for err == nil {
		var c net.Conn
		c, err = n.Accept()
		if err == nil {
			log.Debugf("got inbound bittorrent connection from %s", c.RemoteAddr())
			go sw.inboundConn(c)
		}
	}
	if sw.Running() {
		log.Warn("network lost")
		// suspend torrent's network on abbrupt break
		sw.Torrents.ForEachTorrent(func(t *Torrent) {
			t.LostNetwork()
		})
	}
	sw.net = nil
	return
}

// create a new swarm using a storage backend for storing downloads and torrent metadata
func NewSwarm(storage storage.Storage, gnutella *gnutella.Swarm) *Swarm {
	sw := &Swarm{
		Torrents: Holder{
			st:       storage,
			torrents: make(map[string]*Torrent),
		},
		trackers: map[string]tracker.Announcer{},
		gnutella: gnutella,
	}
	sw.id = common.GeneratePeerID()
	log.Infof("generated peer id %s", sw.id.String())
	return sw
}

// AddOpenTracker adds an opentracker by url to be used by this swarm
func (sw *Swarm) AddOpenTracker(url string) {
	tr := tracker.FromURL(url)
	if tr != nil {
		name := tr.Name()
		_, ok := sw.trackers[name]
		if !ok {
			sw.trackers[name] = tr
		}
	}

}

// implements io.Closer
func (sw *Swarm) Close() (err error) {
	if !sw.closing {
		sw.closing = true
		log.Info("Swarm closing")
		err = sw.Torrents.Close()
	}
	return
}

func (sw *Swarm) AddRemoteTorrent(url string) (err error) {
	sw.WaitForNetwork()
	cl := &http.Client{
		Transport: &http.Transport{
			Dial: sw.net.Dial,
		},
	}
	var info metainfo.TorrentFile
	var r *http.Response
	log.Infof("fetching torrent from %s", url)
	r, err = cl.Get(url)
	if err == nil {
		if r.StatusCode == http.StatusOK {
			defer r.Body.Close()
			err = info.BDecode(r.Body)
			if err == nil {
				var t storage.Torrent
				t, err = sw.Torrents.st.OpenTorrent(&info)
				if err == nil {
					t.VerifyAll(true)
					sw.AddTorrent(t)
				}
			}
		}
	}
	if err != nil {
		log.Errorf("failed to fetch: %s", err.Error())
	}
	return
}
