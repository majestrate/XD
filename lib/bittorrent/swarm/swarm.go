package swarm

import (
	"bytes"
	"github.com/majestrate/XD/lib/bittorrent"
	"github.com/majestrate/XD/lib/bittorrent/extensions"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/dht"
	"github.com/majestrate/XD/lib/gnutella"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/metainfo"
	"github.com/majestrate/XD/lib/network"
	"github.com/majestrate/XD/lib/storage"
	"github.com/majestrate/XD/lib/tracker"
	"github.com/majestrate/XD/lib/util"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// a bittorrent swarm tracking many torrents
type Swarm struct {
	closing  bool
	Torrents Holder
	id       common.PeerID
	trackers map[string]tracker.Announcer
	xdht     dht.XDHT
	gnutella *gnutella.Swarm
	active   int
	getNet   chan network.Network
	netDied  chan bool
	newNet   chan network.Network
	netError chan error
	netDead  bool
}

func (sw *Swarm) IsOnline() bool {
	return !sw.netDead
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
		// check if we should accept this new peer or not
		if !t.ShouldAcceptNewPeer() {
			c.Close()
			return
		}
		var opts extensions.Message
		if h.Reserved.Has(bittorrent.Extension) {
			opts = t.defaultOpts.Copy()
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
	log.Info("Swarm netLoop starting")
	var n network.Network
	for sw.Running() {
		select {
		case newnet := <-sw.newNet:
			log.Info("Network context obtained")
			n = newnet
			sw.netDead = false
		case _ = <-sw.netDied:
			n = nil
			log.Info("Network lost")
			sw.netDead = true
		default:
			if n != nil {
				sw.getNet <- n
			} else {
				log.Debug("network is dead, press 'F' to pay respec")
				time.Sleep(time.Second)
			}
		}
	}
	log.Info("Swarm netLoop exiting")
}

// run until error
func (sw *Swarm) Run() error {
	ticker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-ticker.C:
			sw.tick()
		case err := <-sw.netError:
			ticker.Stop()
			return err
		}
	}
	return nil
}

func (sw *Swarm) tick() {
	sw.Torrents.ForEachTorrent(func(t *Torrent) {
		t.tick()
	})
}

func (sw *Swarm) acceptLoop() {
	for sw.Running() {
		n := <-sw.getNet
		c, err := n.Accept()
		if err == nil {
			log.Debugf("got inbound bittorrent connection from %s", c.RemoteAddr())
			go sw.inboundConn(c)
		} else {
			log.Warnf("failed to accept inbound connection: %s", err.Error())
			sw.netError <- err
			time.Sleep(time.Second)
		}
	}
}

// inform that we lost the network context
func (sw *Swarm) LostNetwork() {
	sw.netDied <- true
}

// give this swarm a new network context
func (sw *Swarm) ObtainedNetwork(n network.Network) {
	sw.id = common.GeneratePeerID()
	log.Infof("Generated new peer id: %s", sw.id.String())
	// give network to netLoop
	sw.newNet <- n
	log.Info("Swarm got network context")
	return
}

// create a new swarm using a storage backend for storing downloads and torrent metadata
func NewSwarm(storage storage.Storage, gnutella *gnutella.Swarm) *Swarm {
	sw := &Swarm{
		Torrents: Holder{
			st: storage,
		},
		trackers: map[string]tracker.Announcer{},
		gnutella: gnutella,
		getNet:   make(chan network.Network),
		newNet:   make(chan network.Network),
		netDied:  make(chan bool),
		netError: make(chan error),
	}
	go sw.acceptLoop()
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
		sw.Torrents.Close(!sw.netDead)
	}
	return
}

func (sw *Swarm) AddRemoteTorrent(remote string) (err error) {
	var u *url.URL
	u, err = url.Parse(remote)
	if err == nil {
		scheme, path := util.SchemePath(u)
		if scheme == "magnet" {
			err = sw.AddMagnet(remote)
		} else if scheme == "file" || scheme == "" {
			err = sw.addFileTorrent(path)
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
			log.Infof("fetched torrent from %s, starting allocation", path)
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
