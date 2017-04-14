package rpc

import (
	"errors"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"xd/lib/bittorrent/swarm"
	"xd/lib/common"
	"xd/lib/log"
)

var ErrNoTorrent = errors.New("no such torrent")

// Bittorrent Swarm RPC Handler
type Server struct {
	sw *swarm.Swarm
	r  *rpc.Server
}

func (r *Server) Register() error {
	r.r = rpc.NewServer()
	return r.r.RegisterName(RPCName, r)
}

func (r *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		if req.URL.Path == RPCPath {
			w.Header().Set("Content-Type", "text/json; encoding=UTF-8")
			c := jsonrpc.NewServerCodec(&rpcIO{
				w: w,
				r: req.Body,
			})
			r.r.ServeRequest(c)
			c.Close()
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *Server) ListTorrents(limit *int, swarms *swarm.TorrentsList) (err error) {
	r.sw.Torrents.ForEachTorrent(func(t *swarm.Torrent) {
		swarms.Infohashes = append(swarms.Infohashes, t.MetaInfo().Infohash().Hex())
	})
	return
}

func (r *Server) TorrentStatus(infohash *string, status *swarm.TorrentStatus) (err error) {
	var ih common.Infohash
	ih, err = common.DecodeInfohash(*infohash)
	if err == nil {
		log.Debugf("getting by infohash: %s ", *infohash)
		r.sw.Torrents.VisitTorrent(ih, func(t *swarm.Torrent) {
			log.Debugf("got torrent by infohash: %s ", *infohash)
			if t == nil {
				err = ErrNoTorrent
			} else {
				*status = t.GetStatus()
			}
		})
	}
	return
}
