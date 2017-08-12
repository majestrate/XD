package rpc

import (
	"encoding/json"
	"xd/lib/bittorrent/swarm"
	"xd/lib/common"
	"xd/lib/log"
)

type TorrentStatusRequest struct {
	Infohash string `json:"infohash"`
}

func (r *TorrentStatusRequest) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	var status swarm.TorrentStatus
	var ih common.Infohash
	var err error
	ih, err = common.DecodeInfohash(r.Infohash)
	if err == nil {
		log.Debugf("getting by infohash: %s ", r.Infohash)
		sw.Torrents.VisitTorrent(ih, func(t *swarm.Torrent) {
			log.Debugf("got torrent by infohash: %s ", r.Infohash)
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

func (ltr *TorrentStatusRequest) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		ParamMethod:   RPCTorrentStatus,
		ParamInfohash: ltr.Infohash,
	})
	return
}
