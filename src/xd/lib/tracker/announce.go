package tracker

import (
	"net"
	"net/url"
	"xd/lib/common"
	"xd/lib/network"
)

type Request struct {
	Infohash common.Infohash
	PeerID common.PeerID
	IP net.Addr
	Port int
	Uploaded int
	Downloaded int
	Left int
	Event string
	NumWant int
	Compact bool
}

type Response struct {
	Interval int `bencode:"interval"`
	Peers []*common.Peer `bencode:"peers"`
}


type Announcer interface {
	// announce and get peers
	Announce(req *Request) (*Response, error)
}


// get announcer from url
// returns nil if invalid url
func FromURL(n network.Network, str string) Announcer {
	u, err := url.Parse(str)
	if err == nil {
		if u.Scheme == "http" {
			return NewHttpTracker(n, str)
		}
	}
	return nil
}
