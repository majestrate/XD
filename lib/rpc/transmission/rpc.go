package transmission

import (
	"encoding/json"
	"fmt"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/sync"
	"github.com/majestrate/XD/lib/util"
	"io"
	"net"
	"net/http"
)

type Server struct {
	sw        *swarm.Swarm
	tokens    sync.Map
	nextToken *xsrfToken
	handlers  map[string]Handler
}

func (s *Server) Error(w http.ResponseWriter, err error, tag Tag) {
	w.WriteHeader(http.StatusOK)
	log.Warnf("trpc error: %s", err.Error())
	json.NewEncoder(w).Encode(Response{
		Tag:    tag,
		Result: err.Error(),
	})
}

func (s *Server) getToken(addr string) *xsrfToken {
	a, _, _ := net.SplitHostPort(addr)
	if a != "" {
		addr = a
	}
	tok, loaded := s.tokens.LoadOrStore(addr, s.nextToken)
	if !loaded {
		s.nextToken = newToken()
	}
	return tok.(*xsrfToken)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	xsrf := r.Header.Get(XSRFToken)
	tok := s.getToken(r.RemoteAddr)
	if !tok.Check(xsrf) {
		tok.Update()
		w.Header().Set(XSRFToken, tok.Token())
		w.WriteHeader(http.StatusConflict)
		return
	}
	tok.Update()
	var req Request
	var resp Response
	err := json.NewDecoder(r.Body).Decode(&req)
	if err == nil {
		log.Debugf("trpc request: %q", req)
		h, ok := s.handlers[req.Method]
		if ok {
			resp = h(s.sw, req.Args)
			if resp.Result != Success {
				log.Warnf("trpc handler non success: %s", resp.Result)
			}
		}
		resp.Tag = req.Tag
	}
	if err == nil {
		buff := new(util.Buffer)
		w.Header().Set("Content-Type", ContentType)
		json.NewEncoder(buff).Encode(resp)
		log.Debugf("trpc response: %s", buff.String())
		w.Header().Set("Content-Length", fmt.Sprintf("%d", buff.Len()))
		io.Copy(w, buff)
	} else {
		s.Error(w, err, req.Tag)
	}
	r.Body.Close()
}

func NewHandler(sw *swarm.Swarm) http.Handler {
	return &Server{
		sw:        sw,
		nextToken: newToken(),
		handlers: map[string]Handler{
			"torrent-start":        NotImplemented,
			"torrent-start-now":    NotImplemented,
			"torrent-stop":         NotImplemented,
			"torrent-verify":       NotImplemented,
			"torrent-reannounce":   NotImplemented,
			"torrent-get":          TorrentGet,
			"torrent-set":          NotImplemented,
			"torrent-add":          NotImplemented,
			"torrent-remove":       NotImplemented,
			"torrent-set-location": NotImplemented,
			"torrent-rename-path":  NotImplemented,
			"session-set":          NotImplemented,
			"session-stats":        NotImplemented,
			"blocklist-update":     NotImplemented,
			"port-test":            NotImplemented,
			"session-close":        NotImplemented,
			"queue-move-top":       NotImplemented,
			"queue-move-up":        NotImplemented,
			"queue-move-down":      NotImplemented,
			"queue-move-bottom":    NotImplemented,
			"free-space":           NotImplemented,
		},
	}
}
