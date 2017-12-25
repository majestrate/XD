package tracker

import (
	"errors"
	"fmt"
	"github.com/zeebo/bencode"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/network"
)

var ErrFailedToTLS = errors.New("failed to tls")

// http tracker
type HttpTracker struct {
	u *url.URL
	// last time we resolved the remote address
	lastResolved time.Time
	// cached network address of tracker
	addr net.Addr
	// how often to resolve network address
	resolveInterval time.Duration
	// currently resolving the address ?
	resolving sync.Mutex
}

// create new http tracker from url
func NewHttpTracker(u *url.URL) *HttpTracker {
	t := &HttpTracker{
		u:               u,
		resolveInterval: time.Hour,
		lastResolved:    time.Unix(0, 0),
	}

	return t
}

func NewHttpsTracker(u *url.URL) *HttpTracker {
	return &HttpTracker{
		u:               u,
		resolveInterval: time.Hour,
		lastResolved:    time.Unix(0, 0),
	}
}

func (t *HttpTracker) shouldResolve() bool {
	return t.lastResolved.Add(t.resolveInterval).Before(time.Now())
}

func (t *HttpTracker) IsOnion() bool {
	return strings.HasSuffix(t.u.Host, ".onion")
}

func (t *HttpTracker) IsI2P() bool {
	return strings.HasSuffix(t.u.Host, ".i2p")
}

// http compact response
type compactHttpAnnounceResponse struct {
	Peers    interface{} `bencode:"peers"`
	Interval int         `bencode:"interval"`
	Error    string      `bencode:"failure reason"`
}

func (t *HttpTracker) Name() string {
	return t.u.String()
}

func (t *HttpTracker) Resolve(n network.Network) (a net.Addr, e error) {
	t.resolving.Lock()
	if t.shouldResolve() {
		var uh string
		uh = t.u.Host
		// XXX: hack
		if strings.Index(uh, ":") == -1 {
			uh += fmt.Sprintf(":%d", t.DefaultPort())
		} else if strings.HasSuffix(uh, ":") {
			uh += fmt.Sprintf("%d", t.DefaultPort())
		}
		log.Debugf("resolve %s", uh)
		parts := strings.Split(uh, ":")
		a, e = n.Lookup(parts[0], parts[1])
		if e == nil {
			t.addr = a
			t.lastResolved = time.Now()
		}
	} else {
		a = t.addr
	}
	t.resolving.Unlock()
	return
}

func (t *HttpTracker) DefaultPort() int {
	if t.IsI2P() {
		return 80
	}
	return 443
}

// send announce via http request
func (t *HttpTracker) Announce(req *Request) (resp *Response, err error) {
	//if req == nil {
	//	return
	//}
	// http client
	var client http.Client

	client.Transport = &http.Transport{
		Dial: func(_, _ string) (c net.Conn, e error) {
			var a net.Addr
			a, e = t.Resolve(req.GetNetwork())
			if e == nil {
				c, e = req.GetNetwork().Dial(a.Network(), a.String())
			}
			return
		},
		DialTLS: func(_, _ string) (c net.Conn, e error) {
			if t.IsOnion() {
				var a net.Addr
				a, e = t.Resolve(req.GetNetwork())
				if e == nil {
					c, e = req.GetNetwork().Dial(a.Network(), a.String())
				}
			} else {
				e = ErrFailedToTLS
			}
			return
		},
	}

	resp = new(Response)
	interval := 30
	// build query
	var u *url.URL
	u, err = url.Parse(t.u.String())
	if err == nil {
		v := u.Query()
		n := req.GetNetwork()
		a := n.Addr()
		var addr string
		if t.IsI2P() {
			addr = a.String() + ".i2p"
		} else if t.IsOnion() {
			var p string
			addr, p, _ = net.SplitHostPort(a.String())
			req.Port, _ = strconv.Atoi(p)
		} else {
			// invalid
		}
		v.Add("ip", addr)
		v.Add("info_hash", string(req.Infohash.Bytes()))
		v.Add("peer_id", string(req.PeerID.Bytes()))
		v.Add("port", fmt.Sprintf("%d", req.Port))
		v.Add("numwant", fmt.Sprintf("%d", req.NumWant))
		v.Add("left", fmt.Sprintf("%d", req.Left))
		if req.Event != Nop {
			v.Add("event", req.Event.String())
		}
		v.Add("downloaded", fmt.Sprintf("%d", req.Downloaded))
		v.Add("uploaded", fmt.Sprintf("%d", req.Uploaded))

		// compact response
		if t.IsOnion() {
			req.Compact = false
			v.Add("compact", "0")
		} else if req.Compact || u.Path != "/a" {
			req.Compact = true
			v.Add("compact", "1")
		}
		u.RawQuery = v.Encode()
		var r *http.Response
		log.Debugf("%s announcing", t.Name())
		r, err = client.Get(u.String())
		if err == nil {
			defer r.Body.Close()
			dec := bencode.NewDecoder(r.Body)
			if req.Compact {
				cresp := new(compactHttpAnnounceResponse)
				err = dec.Decode(cresp)
				if err == nil {
					interval = cresp.Interval
					var cpeers string

					_, ok := cresp.Peers.(string)
					if ok {
						cpeers = cresp.Peers.(string)
						sz := 32
						if t.IsOnion() {
							sz = 12
						}
						l := len(cpeers) / sz
						for l > 0 {
							var p common.Peer
							// TODO: bounds check
							p.Compact = make([]byte, sz)
							copy(p.Compact[:], cpeers[(l-1)*sz:l*sz])
							resp.Peers = append(resp.Peers, p)
							l--
						}
					} else {
						fullpeers, ok := cresp.Peers.([]interface{})
						if ok {
							for idx := range fullpeers {
								// XXX: this is horribad :DDDDDDDDD
								var peer map[string]interface{}
								peer, ok = fullpeers[idx].(map[string]interface{})
								if ok {
									var p common.Peer
									p.IP = fmt.Sprintf("%s", peer["ip"])
									resp.Peers = append(resp.Peers, p)
								}
							}
						}
					}

					if len(cresp.Error) > 0 {
						err = errors.New(cresp.Error)
					}
				}
			} else {
				// decode non compact response
				err = dec.Decode(resp)
				interval = resp.Interval
				if len(resp.Error) > 0 {
					err = errors.New(resp.Error)
				}
			}
		}
	}

	if err == nil {
		log.Infof("%s got %d peers for %s", t.Name(), len(resp.Peers), req.Infohash.Hex())
	} else {
		log.Warnf("%s got error while announcing: %s", t.Name(), err)
	}
	if interval == 0 {
		interval = 60
	}
	resp.NextAnnounce = time.Now().Add(time.Second * time.Duration(interval))
	return
}
