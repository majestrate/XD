package rpc

import (
	"encoding/json"
	"xd/lib/bittorrent/swarm"
)

type ListTorrentsRequest struct {
}

func (ltr *ListTorrentsRequest) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	var swarms swarm.TorrentsList
	sw.Torrents.ForEachTorrent(func(t *swarm.Torrent) {
		swarms.Infohashes = append(swarms.Infohashes, t.MetaInfo().Infohash().Hex())
	})
	w.Return(swarms)
}

func (ltr *ListTorrentsRequest) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		ParamMethod: RPCListTorrents,
	})
	return
}
