package rpc

import (
	"encoding/json"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
)

type AddTorrentRequest struct {
	BaseRequest
	URL string `json:"url"`
}

func (atr *AddTorrentRequest) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	err := sw.AddRemoteTorrent(atr.URL)
	if err == nil {
		w.Return(map[string]interface{}{"error": nil})
	} else {
		w.Return(map[string]interface{}{"error": err.Error()})
	}
}

func (atr *AddTorrentRequest) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		ParamSwarm:  atr.Swarm,
		ParamURL:    atr.URL,
		ParamMethod: RPCAddTorrent,
	})
	return
}
