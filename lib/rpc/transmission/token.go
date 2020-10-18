package transmission

import (
	"time"
	"github.com/majestrate/XD/lib/util"
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
