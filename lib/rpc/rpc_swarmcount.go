package rpc

import (
	"encoding/json"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
)

type SwarmCountRequest struct {
	BaseRequest
	N int
}

func (scr *SwarmCountRequest) ProcessRequest(_ *swarm.Swarm, w *ResponseWriter) {
	w.Return(map[string]interface{}{
		ParamSwarms: scr.N,
	})
}

func (scr *SwarmCountRequest) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		ParamSwarms: scr.N,
		ParamMethod: RPCSwarmCount,
	})
	return
}
