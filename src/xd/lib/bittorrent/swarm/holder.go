package swarm

import (
	"sync"
	"xd/lib/common"
	"xd/lib/storage"
)

// torrent swarm container
type Holder struct {
	closing  bool
	st       storage.Storage
	access   sync.Mutex
	torrents map[string]*Torrent
}

func (h *Holder) addTorrent(t storage.Torrent) {
	if h.closing {
		return
	}
	tr := newTorrent(t)
	h.access.Lock()
	h.torrents[t.Infohash().Hex()] = tr
	h.access.Unlock()
}

func (h *Holder) forEachTorrent(visit func(*Torrent), fork bool) {
	if h.torrents == nil {
		return
	}
	var torrents []*Torrent
	h.access.Lock()
	for _, t := range h.torrents {
		torrents = append(torrents, t)
	}
	h.access.Unlock()
	for _, t := range torrents {
		if fork {
			go visit(t)
		} else {
			visit(t)
		}
	}
}

func (h *Holder) ForEachTorrent(visit func(*Torrent)) {
	h.forEachTorrent(visit, false)
}

func (h *Holder) ForEachTorrentParallel(visit func(*Torrent)) {
	h.forEachTorrent(visit, true)
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

// implements io.Closer
func (h *Holder) Close() (err error) {
	if h.closing {
		return
	}
	var torrents []string
	h.access.Lock()
	h.closing = true
	for n := range h.torrents {
		torrents = append(torrents, n)
	}
	h.access.Unlock()
	for _, n := range torrents {
		go func(name string) {
			h.access.Lock()
			t := h.torrents[name]
			delete(h.torrents, name)
			h.access.Unlock()
			t.Close()
		}(n)
	}
	return
}
