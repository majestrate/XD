package tor

import (
	"crypto/rsa"
	"fmt"
	"github.com/yawning/bulb/utils/pkcs1"
)

type OnionAddr struct {
	k rsa.PublicKey
	p int
}

func (a *OnionAddr) Network() string {
	return "tcp"
}

func (a *OnionAddr) Onion() string {
	id, _ := pkcs1.OnionAddr(&a.k)
	return id
}

func (a *OnionAddr) String() string {
	return fmt.Sprintf("%s.onion:%d", a.Onion(), a.p)
}
