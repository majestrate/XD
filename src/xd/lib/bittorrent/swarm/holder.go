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
	torrents map[common.Infohash]*Torrent
}

func (h *Holder) addTorrent(t storage.Torrent) {
	h.access.Lock()
	defer h.access.Unlock()
	h.torrents[t.Infohash()] = &Torrent{
		st:    t,
		bf:    t.Bitfield(),
		piece: make(chan pieceEvent, 8),
	}
}

func (h *Holder) ForEachTorrent(visit func(*Torrent)) {
	h.access.Lock()
	defer h.access.Unlock()
	for _, t := range h.torrents {
		visit(t)
	}
}

// find a torrent by infohash
// returns nil if we don't have a torrent with this infohash
func (h *Holder) GetTorrent(ih common.Infohash) (t *Torrent) {
	h.access.Lock()
	defer h.access.Unlock()
	t, _ = h.torrents[ih]
	return
}
