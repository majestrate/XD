package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
	"github.com/majestrate/XD/lib/rpc/assets"
	"github.com/majestrate/XD/lib/rpc/transmission"
	"net"
	"net/http"
	"strconv"
)

const ParamMethod = "method"
const ParamSwarm = "swarm"

var ErrNoTorrent = errors.New("no such torrent")

const RPCContentType = "text/json; encoding=UTF-8"

// Bittorrent Swarm RPC Handler
type Server struct {
	sw           []*swarm.Swarm
	fileserver   http.Handler
	expectedHost string
	trpc         http.Handler
}

func NewServer(sw []*swarm.Swarm, host string) *Server {
	fs := assets.GetAssets()
	trpc := transmission.NewHandler(sw[0])
	if fs == nil {
		return &Server{
			sw:           sw,
			expectedHost: host,
			trpc:         trpc,
		}
	} else {
		return &Server{
			sw:           sw,
			expectedHost: host,
			fileserver:   http.FileServer(fs),
			trpc:         trpc,
		}
	}
}

func (r *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if r.expectedHost != "" {
		host := req.Host

		h, _, err := net.SplitHostPort(host)
		if err == nil {
			host = h
		}

		if !(host == r.expectedHost || host == "localhost") {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "expected host %s but got %s", r.expectedHost, host)
			return
		}
	}

	if req.Method == "GET" && r.fileserver != nil {
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
					case RPCSwarmCount:
						rr = &SwarmCountRequest{
							N: len(r.sw),
						}
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
					case RPCListTorrentStatus:
						rr = &ListTorrentStatusRequest{}
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
					if r.sw[swarmidx].IsOnline() {
						rr.ProcessRequest(r.sw[swarmidx], rw)
					} else {
						rr = &rpcError{
							message: "swarm offline",
						}
						rr.ProcessRequest(nil, rw)
					}
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
		} else if req.URL.Path == transmission.RPCPath && r.trpc != nil {
			r.trpc.ServeHTTP(w, req)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
