package transmission

import (
	"encoding/json"
	"net/http"
	"xd/lib/bittorrent/swarm"
)

const RPCPath = "/transmission/rpc"

type Server struct {
	swarms []*swarm.Swarm
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req Request
	json.NewDecoder(r.Body).Decode(&req)
	r.Body.Close()
}

func New(sw []*swarm.Swarm) *Server {
	return &Server{
		swarms: sw,
	}
}
