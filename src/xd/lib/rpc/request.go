package rpc

import (
	"encoding/json"
	"xd/lib/bittorrent/swarm"
	"xd/lib/common"
	"xd/lib/log"
)

type Request interface {
	ProcessRequest(sw *swarm.Swarm, w *ResponseWriter)
	MarshallJSON() ([]byte, error)
}

type TorrentStatusRequest struct {
	Infohash string
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

func (ltr *TorrentStatusRequest) MarshallJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		"method": RPCListTorrents,
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

func (ltr *ListTorrentsRequest) MarshallJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		"method": RPCListTorrents,
	})
	return
}
