package tracker

import (
	"net"
	"net/http"
)

type Tracker struct {
}

func (t *Tracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}
