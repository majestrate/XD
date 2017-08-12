package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"xd/lib/bittorrent/swarm"
)

const ParamMethod = "method"

var ErrNoTorrent = errors.New("no such torrent")

const RPCContentType = "text/json; encoding=UTF-8"

// Bittorrent Swarm RPC Handler
type Server struct {
	sw *swarm.Swarm
}

func NewServer(sw *swarm.Swarm) *Server {
	return &Server{
		sw: sw,
	}
}

func (r *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		if req.URL.Path == RPCPath {
			defer req.Body.Close()
			w.Header().Set("Content-Type", RPCContentType)
			var body map[string]interface{}
			err := json.NewDecoder(req.Body).Decode(&body)
			rw := &ResponseWriter{
				w: w,
			}
			if err == nil {
				var rr Request
				method := body[ParamMethod]
				switch method {
				case RPCListTorrents:
					rr = &ListTorrentsRequest{}
				case RPCTorrentStatus:
					rr = &TorrentStatusRequest{
						Infohash: fmt.Sprintf("%s", body[ParamInfohash]),
					}
				case RPCAddTorrent:
					rr = &AddTorrentRequest{
						URL: fmt.Sprintf("%s", body[ParamURL]),
					}
				default:
					rr = &rpcError{
						message: fmt.Sprintf("no such method %s", method),
					}
				}
				rr.ProcessRequest(r.sw, rw)
			} else {
				// TODO: whatever fix this later
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
