package tracker

import (
	"fmt"
	"time"
	"net/http"
	"net/url"
	"xd/lib/common"
	"xd/lib/network"
	"xd/lib/log"
	"github.com/zeebo/bencode"
)

// http tracker
type HttpTracker struct {
	url string
	session network.Network
	// http client
	client *http.Client
	// next announce
	next time.Time
	// announcing right now?
	announcing bool
}

// create new http tracker from url
func NewHttpTracker(n network.Network, url string) *HttpTracker {
	return &HttpTracker{
		url: url,
		session: n,
		client: &http.Client{
			Transport: &http.Transport{
				Dial: n.Dial,
			},
		},
	}
}

// http compact response
type compactHttpAnnounceResponse struct {
	Peers []byte `bencode:"peers"`
	Interval int `bencode:"interval"`
}

func (t *HttpTracker) Name() string {
	u, err := url.Parse(t.url)
	if err == nil {
		return u.Host
	}
	return t.url
}

// send announce via http request
func (t *HttpTracker) Announce(req *Request) (resp *Response, err error) {
	interval := 30
	var u *url.URL
	u, err = url.Parse(t.url)
	if err == nil {
		
		// build query
		v := u.Query()
		v.Add("ip", req.IP.String())
		v.Add("infohash", string(req.Infohash.Bytes()))
		v.Add("peer_id", string(req.PeerID.Bytes()))
		v.Add("port", fmt.Sprintf("%d", req.Port))
		v.Add("numwant", fmt.Sprintf("%d", req.NumWant))
		if len(req.Event) > 0 {
			v.Add("event", req.Event)
		}
		v.Add("downloaded", fmt.Sprintf("%d", req.Downloaded))
		v.Add("uploaded", fmt.Sprintf("%d", req.Uploaded))
		
		// compact response
		if req.Compact {
			v.Add("compact", "1")
		}
		u.RawQuery = v.Encode()
		var r *http.Response
		t.announcing = true
		log.Debugf("%s announcing", t.Name())
		r, err = t.client.Get(u.String())
		if err == nil {
			defer r.Body.Close()
			dec := bencode.NewDecoder(r.Body)
			resp = new(Response)
			if req.Compact {
				cresp := new(compactHttpAnnounceResponse)
				err = dec.Decode(cresp)
				if err == nil {
					interval = cresp.Interval
					l := len(cresp.Peers) / 32
					for l > 0 {
						p := new(common.Peer)
						// TODO: bounds check
						copy(p.Compact[:], cresp.Peers[(l-1)*32: l*32])
						resp.Peers = append(resp.Peers, p)
						l --
					}
				}
			} else {
				// decode non compact response
				err = dec.Decode(resp)
				interval = resp.Interval
			}
		}
	}
	t.next = time.Now().Add(time.Second * time.Duration(interval))
	log.Infof("%s next announce %s", t.Name(), t.next)
	t.announcing = false
	return
}

func (t *HttpTracker) ShouldAnnounce() bool {
	return time.Now().After(t.next) && !t.announcing
}
