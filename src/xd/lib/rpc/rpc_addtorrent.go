package rpc

import (
	"encoding/json"
	"xd/lib/bittorrent/swarm"
)

type AddTorrentRequest struct {
	URL string `json:"url"`
}

func (atr *AddTorrentRequest) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	sw.AddRemoteTorrent(atr.URL)
	w.Return(map[string]interface{}{"error": nil})
}

func (atr *AddTorrentRequest) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		ParamURL: atr.URL,
	})
	return
}
