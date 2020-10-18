package rpc

import (
	"encoding/json"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
)

type ListTorrentsRequest struct {
	BaseRequest
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
		ParamSwarm:  ltr.Swarm,
		ParamMethod: RPCListTorrents,
	})
	return
}
