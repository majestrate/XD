package tracker

import (
	"fmt"
	"net/http"
	"net/url"
	"xd/lib/common"
	"xd/lib/i2p"
	"github.com/zeebo/bencode"
)

// http tracker
type HttpTracker struct {
	url string
	session i2p.Session
	// http client
	client *http.Client
}

// create new http tracker from url
func NewHttpTracker(s i2p.Session, url string) *HttpTracker {
	return &HttpTracker{
		url: url,
		session: s,
		client: &http.Client{
			Transport: &http.Transport{
				Dial: s.Dial,
			},
		},
	}
}

// http compact response
type compactHttpAnnounceResponse struct {
	Peers []byte `bencode:"peers"`
	Interval int `bencode:"interval"`
}

// send announce via http request
func (t *HttpTracker) Announce(req *Request) (resp *Response, err error) {
	var u *url.URL
	u, err = url.Parse(t.url)
	if err == nil {
		// we connected
		
		// build query
		v := u.Query()
		v.Add("ip", req.IP.String())
		v.Add("infohash", string(req.Infohash.Bytes()))
		v.Add("peer_id", string(req.PeerID.Bytes()))
		v.Add("port", fmt.Sprintf("%d", req.Port))
		v.Add("numwant", fmt.Sprintf("%d", req.NumWant))
		v.Add("event", req.Event)
		v.Add("downloaded", fmt.Sprintf("%d", req.Downloaded))
		v.Add("uploaded", fmt.Sprintf("%d", req.Uploaded))
		
		// compact response
		if req.Compact {
			v.Add("compact", "1")
		}
		u.RawQuery = v.Encode()
		var r *http.Response
		r, err = t.client.Get(u.String())
		if err == nil {
			defer r.Body.Close()
			dec := bencode.NewDecoder(r.Body)
			resp = new(Response)
			if req.Compact {
				cresp := new(compactHttpAnnounceResponse)
				err = dec.Decode(cresp)
				if err == nil {
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
			}
		}
	}
	return
}
