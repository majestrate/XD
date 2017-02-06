package swarm

import (
	"errors"
	"xd/lib/common"
)

var ErrNoTorrent = errors.New("no such torrent")

type RPC struct {
	sw *Swarm
}

func (r *RPC) TorrentStatus(infohash *string, status *TorrentStatus) (err error) {
	var ih common.Infohash
	err = ih.Decode(*infohash)
	if err == nil {
		t := r.sw.Torrents.GetTorrent(ih)
		if t == nil {
			err = ErrNoTorrent
		} else {
			status = t.GetStatus()
		}
	}
	return
}
