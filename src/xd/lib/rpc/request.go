package rpc

import (
	"xd/lib/bittorrent/swarm"
)

type Request interface {
	// handle request on server
	ProcessRequest(sw *swarm.Swarm, w *ResponseWriter)
	// convert request to json
	MarshalJSON() ([]byte, error)
}
