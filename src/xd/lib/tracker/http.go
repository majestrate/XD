package tracker

import (
	"errors"
	"fmt"
	"github.com/zeebo/bencode"
	"net/http"
	"net/url"
	"time"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/network"
)

// http tracker
type HttpTracker struct {
	url     string
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
		url:     url,
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
	Peers    []byte `bencode:"peers"`
	Interval int    `bencode:"interval"`
	Error    string `bencode:"failure reason"`
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
	t.announcing = true
	resp = new(Response)
	interval := 30
	var u *url.URL
	u, err = url.Parse(t.url)
	if err == nil {

		// build query
		v := u.Query()
		v.Add("ip", req.IP.String()+".i2p")
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
		if req.Compact {
			v.Add("compact", "1")
		}
		u.RawQuery = v.Encode()
		var r *http.Response
		log.Debugf("%s announcing", t.Name())
		r, err = t.client.Get(u.String())
		if err == nil {
			defer r.Body.Close()
			dec := bencode.NewDecoder(r.Body)
			if req.Compact {
				cresp := new(compactHttpAnnounceResponse)
				err = dec.Decode(cresp)
				if err == nil {
					interval = cresp.Interval
					l := len(cresp.Peers) / 32
					for l > 0 {
						var p common.Peer
						// TODO: bounds check
						copy(p.Compact[:], cresp.Peers[(l-1)*32:l*32])
						resp.Peers = append(resp.Peers, p)
						l--
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
	if err != nil {
		log.Warnf("%s got error while announcing: %s", t.Name(), err)
	}
	if interval == 0 {
		interval = 60
	}
	t.next = time.Now().Add(time.Second * time.Duration(interval))
	log.Infof("%s got %d peers, next announce %s (interval was %d)", t.Name(), len(resp.Peers), t.next, interval)
	t.announcing = false
	return
}

func (t *HttpTracker) ShouldAnnounce() bool {
	return time.Now().After(t.next) && !t.announcing
}
