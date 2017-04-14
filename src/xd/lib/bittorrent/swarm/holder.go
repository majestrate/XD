package swarm

import (
	"sync"
	"xd/lib/common"
	"xd/lib/storage"
)

// torrent swarm container
type Holder struct {
	st       storage.Storage
	access   sync.Mutex
	torrents map[string]*Torrent
}

func (h *Holder) addTorrent(t storage.Torrent) {

	tr := newTorrent(t)
	h.access.Lock()
	h.torrents[t.Infohash().Hex()] = tr
	h.access.Unlock()
}

func (h *Holder) ForEachTorrent(visit func(*Torrent)) {
	var torrents []*Torrent
	h.access.Lock()
	for _, t := range h.torrents {
		torrents = append(torrents, t)
	}
	h.access.Unlock()
	for _, t := range torrents {
		visit(t)
	}
}

// find a torrent by infohash
// returns nil if we don't have a torrent with this infohash
func (h *Holder) GetTorrent(ih common.Infohash) (t *Torrent) {
	h.access.Lock()
	t, _ = h.torrents[ih.Hex()]
	h.access.Unlock()
	return
}

func (h *Holder) VisitTorrent(ih common.Infohash, visit func(*Torrent)) {
	var t *Torrent
	h.access.Lock()
	t, _ = h.torrents[ih.Hex()]
	h.access.Unlock()
	visit(t)
}
