package tracker

import (
	"net"
	"net/url"
	"strings"
	"xd/lib/common"
	"xd/lib/network"
)

type Event string

const Started = Event("started")
const Stopped = Event("stopped")
const Completed = Event("completed")
const Nop = Event("")

func (ev Event) String() string {
	return string(ev)
}

type Request struct {
	Infohash   common.Infohash
	PeerID     common.PeerID
	IP         net.Addr
	Port       int
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Event      Event
	NumWant    int
	Compact    bool
}

type Response struct {
	Interval int           `bencode:"interval"`
	Peers    []common.Peer `bencode:"peers"`
	Error    string        `bencode:"failure reason"`
}

// bittorrent announcer, gets peers and announces presence in swarm
type Announcer interface {
	// announce and get peers
	Announce(req *Request) (*Response, error)
	// return true if we should announce otherwise return false
	ShouldAnnounce() bool
	// name of this tracker
	Name() string
}

// get announcer from url
// returns nil if invalid url
func FromURL(n network.Network, str string) Announcer {
	u, err := url.Parse(str)
	if err == nil {
		if u.Scheme == "http" && strings.HasSuffix(u.Host, ".i2p") {
			return NewHttpTracker(n, str)
		}
	}
	return nil
}
