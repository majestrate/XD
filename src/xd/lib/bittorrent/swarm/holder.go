package swarm

import (
	"sync"
	"xd/lib/common"
	"xd/lib/storage"
)

// torrent swarm container
type Holder struct {
	st storage.Storage
	access sync.Mutex
	torrents map[common.Infohash]*Torrent
}

// find a torrent by infohash
// returns nil if we don't have a torrent with this infohash
func (h *Holder) GetTorrent(ih common.Infohash) (t *Torrent) {
	h.access.Lock()
	defer h.access.Unlock()
	t, _ = h.torrents[ih]
	return
}
