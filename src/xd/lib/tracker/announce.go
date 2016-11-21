package tracker

import (
	"net"
	"xd/lib/common"
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
