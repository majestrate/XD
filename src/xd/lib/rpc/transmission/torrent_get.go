package transmission

import (
	"xd/lib/bittorrent/swarm"
)

type tgResp map[string]interface{}

func (t *tgResp) Set(key string, val interface{}) {
	(*t)[key] = val
}

type tgFieldHandler func(*swarm.Swarm, *tgResp, TorrentID) error

func tgHandleID(sw *swarm.Swarm, resp *tgResp, id TorrentID) (err error) {
	resp.Set("id", id)
	return
}

func tgHandleName(sw *swarm.Swarm, resp *tgResp, id TorrentID) (err error) {
	t := sw.Torrents.GetTorrentByID(int64(id))
	if t == nil {
		resp.Set("name", "???")
	} else {
		resp.Set("name", t.Name())
	}
	return
}

var tgFieldHandlers = map[string]tgFieldHandler{
	"id":   tgHandleID,
	"name": tgHandleName,
}
