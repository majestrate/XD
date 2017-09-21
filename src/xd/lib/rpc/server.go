package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"xd/lib/bittorrent/swarm"
	"xd/lib/rpc/assets"
)

const ParamMethod = "method"
const ParamSwarm = "swarm"

var ErrNoTorrent = errors.New("no such torrent")

const RPCContentType = "text/json; encoding=UTF-8"

// Bittorrent Swarm RPC Handler
type Server struct {
	sw         []*swarm.Swarm
	fileserver http.Handler
}

func NewServer(sw []*swarm.Swarm) *Server {
	fs := assets.GetAssets()
	if fs == nil {
		return &Server{
			sw: sw,
		}
	} else {
		return &Server{
			sw:         sw,
			fileserver: http.FileServer(fs),
		}
	}
}

func (r *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" && r.fileserver != nil {
		req.URL.Path = assets.Prefix + req.URL.Path
		r.fileserver.ServeHTTP(w, req)
	} else if req.Method == "POST" {
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
				swarmno, ok := body[ParamSwarm]
				swarmidx := 0
				if ok {
					swarmidx, err = strconv.Atoi(fmt.Sprintf("%s", swarmno))
				}
				if err == nil {
					switch method {
					case RPCChangeTorrent:
						rr = &ChangeTorrentRequest{
							Infohash: fmt.Sprintf("%s", body[ParamInfohash]),
							Action:   fmt.Sprintf("%s", body[ParamAction]),
						}
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
					case RPCSetPieceWindow:
						n, ok := body[ParamN].(float64)
						if ok {
							rr = &SetPieceWindowRequest{
								N: int(n),
							}
						} else {
							rr = &rpcError{
								message: fmt.Sprintf("invalid value: %s", body[ParamN]),
							}
						}
					default:
						rr = &rpcError{
							message: fmt.Sprintf("no such method %s", method),
						}
					}
				} else {
					rr = &rpcError{
						message: err.Error(),
					}
				}
				if swarmidx < len(r.sw) {
					rr.ProcessRequest(r.sw[swarmidx], rw)
				} else {
					rr = &rpcError{
						message: "no such swarm",
					}
					rr.ProcessRequest(nil, rw)
				}
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
