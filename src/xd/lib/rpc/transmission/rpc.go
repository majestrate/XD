package transmission

import (
	"encoding/json"
	"net/http"
	"time"
	"xd/lib/bittorrent/swarm"
	"xd/lib/sync"
	"xd/lib/util"
)

type xsrfToken struct {
	data    string
	expires time.Time
}

func newToken() *xsrfToken {
	return &xsrfToken{
		data:    util.RandStr(10),
		expires: time.Now().Add(time.Minute),
	}
}

type Server struct {
	swarms    []*swarm.Swarm
	tokens    sync.Map
	nextToken *xsrfToken
}

func (s *Server) Error(w http.ResponseWriter, err error, tag Tag) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Tag:    tag,
		Result: err.Error(),
	})
}

func (t *xsrfToken) Expired() bool {
	return time.Now().After(t.expires)
}

func (t *xsrfToken) Update() {
	if t.Expired() {
		t.Regen()
	}
}

func (t *xsrfToken) Token() string {
	return t.data
}

func (t *xsrfToken) Regen() {
	t.data = util.RandStr(10)
	t.expires = time.Now().Add(time.Minute)
}

func (t *xsrfToken) Check(tok string) bool {
	return t.data == tok && !t.Expired()
}

func (s *Server) getXSRFToken(addr string) *xsrfToken {
	tok, loaded := s.tokens.LoadOrStore(addr, s.nextToken)
	if !loaded {
		s.nextToken = newToken()
	}
	return tok.(*xsrfToken)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	xsrf := r.Header.Get(XSRFToken)
	tok := s.getXSRFToken(r.RemoteAddr)
	if !tok.Check(xsrf) {
		w.Header().Set(XSRFToken, tok.Token())
		w.WriteHeader(http.StatusConflict)
		return
	}
	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err == nil {
	}
	if err != nil {
		s.Error(w, err, req.Tag)
	}
	r.Body.Close()
}

func New(sw []*swarm.Swarm) *Server {
	return &Server{
		swarms:    sw,
		nextToken: newToken(),
	}
}
