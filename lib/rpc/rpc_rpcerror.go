package rpc

import (
	"encoding/json"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
)

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
