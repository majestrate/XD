package swarm

import (
	"errors"
	"xd/lib/common"
	"xd/lib/log"
)

var ErrNoTorrent = errors.New("no such torrent")

// Bittorrent Swarm RPC
type RPC struct {
	sw *Swarm
}

const RPCName = "XD"

const RPCListTorrents = RPCName + ".ListTorrents"

func (r *RPC) ListTorrents(limit *int, swarms *TorrentsList) (err error) {
	r.sw.Torrents.ForEachTorrent(func(t *Torrent) {
		swarms.Infohashes = append(swarms.Infohashes, t.MetaInfo().Infohash().Hex())
	})
	return
}

const RPCTorrentStatus = RPCName + ".TorrentStatus"

func (r *RPC) TorrentStatus(infohash *string, status *TorrentStatus) (err error) {
	var ih common.Infohash
	ih, err = common.DecodeInfohash(*infohash)
	if err == nil {
		log.Debugf("getting by infohash: %s ", *infohash)
		t := r.sw.Torrents.GetTorrent(ih)
		log.Debugf("got torrent by infohash: %s ", *infohash)
		if t == nil {
			err = ErrNoTorrent
		} else {
			st := t.GetStatus()
			status.Peers = st.Peers
		}
	}
	return
}
