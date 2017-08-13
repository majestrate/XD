package rpc

import (
	"encoding/json"
	"xd/lib/bittorrent/swarm"
)

type SetPieceWindowRequest struct {
	N int `json:"n"`
}

func (r *SetPieceWindowRequest) ProcessRequest(sw *swarm.Swarm, w *ResponseWriter) {
	if r.N > 0 {
		sw.Torrents.MaxReq = r.N
		sw.Torrents.ForEachTorrent(func(t *swarm.Torrent) {
			t.MaxRequests = r.N
			t.VisitPeers(func(c *swarm.PeerConn) {
				c.MaxParalellRequests = r.N
			})
		})
		w.Return(map[string]interface{}{"error": nil})
	} else {
		w.SendError("N must be greater than zero")
	}
}

func (r *SetPieceWindowRequest) MarshalJSON() (data []byte, err error) {
	data, err = json.Marshal(map[string]interface{}{
		ParamMethod: RPCSetPieceWindow,
		ParamN:      r.N,
	})
	return
}
