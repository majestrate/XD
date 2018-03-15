package swarm

import (
	"bytes"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	closing   bool
	Torrents  Holder
	id        common.PeerID
	trackers  map[string]tracker.Announcer
	xdht      dht.XDHT
	gnutella  *gnutella.Swarm
	active    int
	getNet    chan network.Network
	netStatus chan bool
	newNet    chan network.Network
}

func (sw *Swarm) Running() bool {
	return !sw.closing
}

func (sw *Swarm) onStopped(t *Torrent) {
	sw.active--
}

func (sw *Swarm) Network() network.Network {
	return <-sw.getNet
}

func (sw *Swarm) waitForQueue() {
	if sw.Torrents.QueueSize > 0 {
		for sw.active >= sw.Torrents.QueueSize {
			time.Sleep(time.Second)
		}
	}
}

func (sw *Swarm) startTorrent(t *Torrent) {
	t.RemoveSelf = func() {
		sw.Torrents.removeTorrent(t.st.Infohash())
	}
	t.Stopped = func() {
		sw.onStopped(t)
	}
	// wait for network
	sw.Network()
	t.xdht = &sw.xdht
	// give peerid
	t.id = sw.id
	// add open trackers
	for name := range sw.trackers {
		t.Trackers[name] = sw.trackers[name]
	}

	info := t.MetaInfo()
	if info != nil {
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
	}
	// handle messages
	sw.waitForQueue()
	sw.active++
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
		var opts extensions.Message
		if h.Reserved.Has(bittorrent.Extension) {
			if t.Ready() {
				opts = extensions.NewOur(uint32(len(t.metaInfo)))
			} else {
				opts = extensions.NewOur(0)
			}
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
		p.inbound = true
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
	sw.Torrents.addTorrent(t, sw.Network)
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

func (sw *Swarm) netLoop() {
	var netStatus bool
	var n network.Network
	for sw.Running() {
		select {
		case n = <-sw.newNet:
			log.Info("new network context obtained")
		case netStatus = <-sw.netStatus:
			if netStatus {
				log.Info("network obtained")
			} else {
				log.Info("network lost")
			}
		default:
			if netStatus {
				sw.getNet <- n
			} else {
				time.Sleep(time.Millisecond * 100)
			}
		}
	}
}

// run with network context
func (sw *Swarm) Run(n network.Network) (err error) {
	// broadcast we have gotten a network context
	sw.netStatus <- true
	// give network to netLoop
	sw.newNet <- n
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
		// regenerate peer id
		sw.id = common.GeneratePeerID()
		sw.netStatus <- false
	}
	return
}

// create a new swarm using a storage backend for storing downloads and torrent metadata
func NewSwarm(storage storage.Storage, gnutella *gnutella.Swarm) *Swarm {
	sw := &Swarm{
		Torrents: Holder{
			st: storage,
		},
		trackers:  map[string]tracker.Announcer{},
		gnutella:  gnutella,
		getNet:    make(chan network.Network),
		newNet:    make(chan network.Network),
		netStatus: make(chan bool),
	}
	sw.id = common.GeneratePeerID()
	log.Infof("generated peer id %s", sw.id.String())
	go sw.netLoop()
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

func (sw *Swarm) AddRemoteTorrent(remote string) (err error) {
	var u *url.URL
	u, err = url.Parse(remote)
	if err == nil {
		scheme := strings.ToLower(u.Scheme)
		if scheme == "magnet" {
			err = sw.AddMagnet(remote)
		} else if scheme == "file" || scheme == "" {
			err = sw.addFileTorrent(u.Path)
		} else {
			err = sw.addHTTPTorrent(u.String())
		}
	}
	return
}

func (sw *Swarm) AddMagnet(uri string) (err error) {
	var u *url.URL
	u, err = url.Parse(uri)
	if err == nil {
		q := u.Query()
		xt := q.Get("xt")
		if len(xt) > 0 {
			xt = strings.ToLower(xt)
			if strings.HasPrefix(xt, "urn:btih:") && len(xt) == 49 {
				var ih common.Infohash
				ih, err = common.DecodeInfohash(xt[9:])
				if err == nil {
					err = sw.addMagnet(ih)
				}
			} else {
				err = common.ErrBadMagnetURI
			}
		} else {
			err = common.ErrBadMagnetURI
		}
	}
	return
}

func (sw *Swarm) addMagnet(ih common.Infohash) (err error) {
	sw.AddTorrent(sw.Torrents.st.EmptyTorrent(ih))
	return
}

func (sw *Swarm) addFileTorrent(path string) (err error) {
	var info metainfo.TorrentFile
	var f *os.File
	f, err = os.Open(path)
	if err == nil {
		err = info.BDecode(f)
		f.Close()
		if err == nil {
			var t storage.Torrent
			t, err = sw.Torrents.st.OpenTorrent(&info)
			if err == nil {
				err = t.VerifyAll()
				if err == nil {
					sw.AddTorrent(t)
				}
			}
		}
	}
	if err != nil {
		log.Errorf("failed to load torrent %s", err.Error())
	}
	return
}

func (sw *Swarm) addHTTPTorrent(remote string) (err error) {
	n := sw.Network()
	cl := &http.Client{
		Transport: &http.Transport{
			Dial: n.Dial,
		},
	}
	var info metainfo.TorrentFile
	var r *http.Response
	log.Infof("fetching torrent from %s", remote)
	r, err = cl.Get(remote)
	if err == nil {
		if r.StatusCode == http.StatusOK {
			defer r.Body.Close()
			err = info.BDecode(r.Body)
			if err == nil {
				var t storage.Torrent
				t, err = sw.Torrents.st.OpenTorrent(&info)
				if err == nil {
					err = t.VerifyAll()
					if err == nil {
						sw.AddTorrent(t)
					}
				}
			}
		}
	}
	if err != nil {
		log.Errorf("failed to fetch torrent: %s", err.Error())
	}
	return
}
