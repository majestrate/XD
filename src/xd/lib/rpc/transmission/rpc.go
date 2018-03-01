package transmission

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"xd/lib/bittorrent/swarm"
	"xd/lib/sync"
	"xd/lib/util"
)

type Server struct {
	sw        *swarm.Swarm
	tokens    sync.Map
	nextToken *xsrfToken
	handlers  map[string]Handler
}

func (s *Server) Error(w http.ResponseWriter, err error, tag Tag) {
	w.WriteHeader(http.StatusOK)
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
		h, ok := s.handlers[req.Method]
		if ok {
			resp = h(req.Args)
		}
		resp.Tag = req.Tag
	}
	if err == nil {
		buff := new(util.Buffer)
		w.Header().Set("Content-Type", ContentType)
		json.NewEncoder(buff).Encode(resp)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", buff.Len()))
		io.Copy(w, buff)
	} else {
		s.Error(w, err, req.Tag)
	}
	r.Body.Close()
}

func New(sw *swarm.Swarm) *Server {
	return &Server{
		sw:        sw,
		nextToken: newToken(),
		handlers:  make(map[string]Handler),
	}
}
