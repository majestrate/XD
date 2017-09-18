package rpc

import (
	"encoding/json"
	"xd/lib/bittorrent/swarm"
)

type AddTorrentRequest struct {
	BaseRequest
	URL string `json:"url"`
}

func (atr *AddTorrentRequest) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	go sw.AddRemoteTorrent(atr.URL)
	w.Return(map[string]interface{}{"error": nil})
}

func (atr *AddTorrentRequest) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		ParamSwarm:  atr.Swarm,
		ParamURL:    atr.URL,
		ParamMethod: RPCAddTorrent,
	})
	return
}
