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
	fails    time.Duration
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
			GetNetwork: a.t.Network,
		}
		if !a.t.Done() {
			req.Left = a.t.st.DownloadRemaining()
		}
		if ev == tracker.Stopped {
			req.NumWant = 0
		}
		var resp *tracker.Response
		resp, err = a.announce.Announce(req)
		backoff := a.fails * time.Minute
		a.next = resp.NextAnnounce.Add(backoff)
		if err == nil {
			a.t.addPeers(resp.Peers)
		}
	}
	a.access.Unlock()
	return
}
