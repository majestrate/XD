package rpc

import (
	"errors"
	"net/http"
	"xd/lib/bittorrent/swarm"
)

var ErrNoTorrent = errors.New("no such torrent")

const RPCContentType = "text/json; encoding=UTF-8"

// Bittorrent Swarm RPC Handler
type Server struct {
	sw *swarm.Swarm
}

func (r *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		if req.URL.Path == RPCPath {
			defer req.Body.Close()
			w.Header().Set("Content-Type", RPCContentType)
			// json.NewDecoder(req.Body).Decode()
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
