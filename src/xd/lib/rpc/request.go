package rpc

import (
	"encoding/json"
	"xd/lib/bittorrent/swarm"
	"xd/lib/common"
	"xd/lib/log"
)

const ParamInfohash = "infohash"

type Request interface {
	// handle request on server
	ProcessRequest(sw *swarm.Swarm, w *ResponseWriter)
	// convert request to json
	MarshalJSON() ([]byte, error)
}

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

type rpcError struct {
	message string
}

func (e *rpcError) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]string{
		"error": e.message,
	})
	return
}

func (e *rpcError) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	w.SendError(e.message)
}
