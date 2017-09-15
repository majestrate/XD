package swarm

import (
	"sync"
	"time"
	"xd/lib/tracker"
)

const DefaultAnnounceNumWant = 10
const DefaultAnnouncePort = 6881

type torrentAnnounce struct {
	access   sync.Mutex
	next     time.Time
	fails    uint32
	announce tracker.Announcer
	t        *Torrent
}

func (a *torrentAnnounce) tryAnnounce(ev tracker.Event) (err error) {
	a.access.Lock()
	if time.Now().After(a.next) {
		req := &tracker.Request{
			Infohash:   a.t.st.Infohash(),
			PeerID:     a.t.id,
			Port:       DefaultAnnouncePort,
			Event:      ev,
			NumWant:    DefaultAnnounceNumWant,
			Left:       a.t.st.DownloadRemaining(),
			GetNetwork: a.t.Network,
		}
		var resp *tracker.Response
		resp, err = a.announce.Announce(req)
		backoff := time.Minute * time.Duration(a.fails)
		a.next = resp.NextAnnounce.Add(backoff)
		if err == nil {
			a.t.addPeers(resp.Peers)
		}
	}
	a.access.Unlock()
	return
}
