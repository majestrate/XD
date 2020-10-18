package tracker

import (
	"net/url"
	"time"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/network"
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
	Port       int
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Event      Event
	NumWant    int
	Compact    bool
	GetNetwork func() network.Network
}

type Response struct {
	Interval     int           `bencode:"interval"`
	Peers        []common.Peer `bencode:"peers"`
	Error        string        `bencode:"failure reason"`
	NextAnnounce time.Time     `bencode:"-"`
}

// bittorrent announcer, gets peers and announces presence in swarm
type Announcer interface {
	// announce and get peers
	Announce(req *Request) (*Response, error)
	// name of this tracker
	Name() string
}

// get announcer from url
// returns nil if invalid url
func FromURL(str string) Announcer {
	u, err := url.Parse(str)
	if err == nil {
		if u.Scheme == "http" {
			return NewHttpTracker(u)
		}
	}
	return nil
}
