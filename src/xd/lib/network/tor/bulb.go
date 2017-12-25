package tor

import (
	"crypto/rsa"
)

func CreateSession(n, a, k, p string, externalPort int) *Session {
	if externalPort == 0 {
		externalPort = 6889
	}
	return &Session{
		net:       n,
		addr:      a,
		keys:      k,
		passwd:    p,
		port:      externalPort,
		subs:      make(map[string]*eventSub),
		nameCache: make(map[string]rsa.PublicKey),
	}
}
