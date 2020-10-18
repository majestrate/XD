package swarm

import (
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/network"
	"github.com/majestrate/XD/lib/storage"
	"github.com/majestrate/XD/lib/sync"
)

// torrent swarm container
type Holder struct {
	closing      bool
	st           storage.Storage
	torrents     sync.Map
	torrentsByID sync.Map
	MaxReq       int
	QueueSize    int
}

func (h *Holder) TorrentIDs() (ids map[int64]string) {
	ids = make(map[int64]string)
	h.ForEachTorrent(func(t *Torrent) {
		ids[t.TID] = t.Infohash().Hex()
	})
	return
}

func (h *Holder) GetTorrentByID(id int64) (t *Torrent) {
	tr, ok := h.torrentsByID.Load(id)
	if ok {
		t = tr.(*Torrent)
	}
	return
}

func (h *Holder) addTorrent(t storage.Torrent, getNet func() network.Network) {
	if h.closing {
		return
	}
	tr := newTorrent(t, getNet)
	tr.MaxRequests = h.MaxReq
	h.torrents.Store(t.Infohash().Hex(), tr)
	h.torrentsByID.Store(tr.TID, tr)
}

func (h *Holder) addMagnet(ih common.Infohash, getNet func() network.Network) {
	if h.closing {
		return
	}
	tr := newTorrent(h.st.EmptyTorrent(ih), getNet)
	tr.MaxRequests = h.MaxReq
	h.torrents.Store(ih.Hex(), tr)
	h.torrentsByID.Store(tr.TID, tr)
}

func (h *Holder) removeTorrent(ih common.Infohash) {
	if h.closing {
		return
	}
	tr, ok := h.torrents.Load(ih.Hex())
	if ok {
		h.torrents.Delete(ih.Hex())
		h.torrentsByID.Delete(tr.(*Torrent).TID)
	}
}

func (h *Holder) forEachTorrent(visit func(*Torrent), fork bool) {
	if h.closing {
		return
	}
	h.torrents.Range(func(_, v interface{}) bool {
		t := v.(*Torrent)
		if fork {
			go visit(t)
		} else {
			visit(t)
		}
		return true
	})
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
	v, ok := h.torrents.Load(ih.Hex())
	if ok {
		t = v.(*Torrent)
	}
	return
}

func (h *Holder) VisitTorrent(ih common.Infohash, visit func(*Torrent)) {
	visit(h.GetTorrent(ih))
}

func (h *Holder) Close(announce bool) {
	if h.closing {
		return
	}
	var wg sync.WaitGroup
	h.closing = true
	h.torrentsByID.Range(func(k, _ interface{}) bool {
		h.torrentsByID.Delete(k)
		return false
	})
	h.torrents.Range(func(k, v interface{}) bool {
		t := v.(*Torrent)
		wg.Add(1)
		go func() {
			if announce {
				t.Stop()
			} else {
				t.StopAnnouncing(false)
				t.Close()
			}
			h.torrents.Delete(k)
			wg.Add(-1)
		}()
		return false
	})
	wg.Wait()
	return
}
