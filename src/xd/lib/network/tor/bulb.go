package tor

import (
	"crypto/rsa"
)

func CreateSession(n, a, k, p string) *Session {
	return &Session{
		net:       n,
		addr:      a,
		keys:      k,
		passwd:    p,
		subs:      make(map[string]*eventSub),
		nameCache: make(map[string]rsa.PublicKey),
	}
}
