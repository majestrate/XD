package rpc

import (
	"xd/lib/bittorrent/swarm"
)

type Request interface {
	ProcessRequest(sw *swarm.Swarm, w *ResponseWriter)
}

type ListTorrentsRequest struct {
}

func (ltr *ListTorrentsRequest) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	var swarms swarm.TorrentsList
	sw.Torrents.ForEachTorrent(func(t *swarm.Torrent) {
		swarms.Infohashes = append(swarms.Infohashes, t.MetaInfo().Infohash().Hex())
	})
	w.Return(swarms)
}
