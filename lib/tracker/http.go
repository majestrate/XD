package tracker

import (
	"errors"
	"fmt"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/sync"
	"github.com/zeebo/bencode"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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

func (t *HttpTracker) shouldResolve() bool {
	return t.lastResolved.Add(t.resolveInterval).Before(time.Now())
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
			t.resolving.Lock()
			if t.shouldResolve() {
				var h, p string
				// XXX: hack
				if strings.Index(t.u.Host, ":") == -1 {
					t.u.Host += ":80"
				}
				h, p, e = net.SplitHostPort(t.u.Host)
				if e == nil {
					a, e = req.GetNetwork().Lookup(h, p)
					if e == nil {
						t.addr = a
						t.lastResolved = time.Now()
					}
				}
			} else {
				a = t.addr
			}
			t.resolving.Unlock()
			if e == nil {
				c, e = req.GetNetwork().Dial(a.Network(), a.String())
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
		host, _, _ := net.SplitHostPort(a.String())
		if a.Network() == "i2p" {
			host += ".i2p"
			req.Compact = true
		}
		v.Add("ip", host)
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
		if req.Compact || u.Path != "/a" {
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
						l := len(cpeers) / 32
						for l > 0 {
							var p common.Peer
							// TODO: bounds check
							copy(p.Compact[:], cpeers[(l-1)*32:l*32])
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
									port, ok := peer["port"].(int64)
									if ok {
										p.Port = int(port)
									}
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
