package rpc

import (
	"encoding/json"
	"errors"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
	"github.com/majestrate/XD/lib/common"
)

const TorrentChangeStart = "start"
const TorrentChangeStop = "stop"
const TorrentChangeRemove = "remove"
const TorrentChangeDelete = "delete"

var ErrInvalidAction = errors.New("invalid torrent action")

type ChangeTorrentRequest struct {
	BaseRequest
	Infohash string `json:"infohash"`
	Action   string `json:"action"`
}

func (r *ChangeTorrentRequest) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	var ih common.Infohash
	var err error
	ih, err = common.DecodeInfohash(r.Infohash)
	if err == nil {
		sw.Torrents.VisitTorrent(ih, func(t *swarm.Torrent) {
			if t == nil {
				err = ErrNoTorrent
			} else {
				switch r.Action {
				case TorrentChangeStart:
					err = t.Start()
				case TorrentChangeStop:
					err = t.Stop()
				case TorrentChangeRemove:
					err = t.Remove()
				case TorrentChangeDelete:
					err = t.Delete()
				default:
					err = ErrInvalidAction
				}
			}
		})
	}
	if err == nil {
		w.Return(map[string]interface{}{"error": nil})
	} else {
		w.Return(map[string]interface{}{"error": err.Error()})
	}
}

func (r *ChangeTorrentRequest) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		ParamSwarm:    r.Swarm,
		ParamInfohash: r.Infohash,
		ParamAction:   r.Action,
		ParamMethod:   RPCChangeTorrent,
	})
	return
}
