package rpc

import (
	"encoding/json"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
	"github.com/majestrate/XD/lib/common"
)

type TorrentStatusRequest struct {
	BaseRequest
	Infohash string `json:"infohash"`
}

func (r *TorrentStatusRequest) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	var status swarm.TorrentStatus
	var ih common.Infohash
	var err error
	ih, err = common.DecodeInfohash(r.Infohash)
	if err == nil {
		sw.Torrents.VisitTorrent(ih, func(t *swarm.Torrent) {
			if t == nil {
				err = ErrNoTorrent
			} else {
				status = t.GetStatus()
			}
		})
	}
	if err == nil {
		w.Return(status)
	} else {
		w.SendError(err.Error())
	}
}

func (r *TorrentStatusRequest) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		ParamSwarm:    r.Swarm,
		ParamMethod:   RPCTorrentStatus,
		ParamInfohash: r.Infohash,
	})
	return
}
