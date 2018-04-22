package i2p

import (
	"crypto/sha256"
	"net"
	"strings"
)

// implements net.Addr
type Addr struct {
	addr string
	port string
}

func (a Addr) Network() string {
	return "i2p"
}

func (a Addr) String() string {
	return net.JoinHostPort(a.addr, a.port)
}

func I2PAddr(addr string) Addr {
	if strings.Count(addr, ":") > 0 {
		a, p, _ := net.SplitHostPort(addr)
		return Addr{
			addr: a,
			port: p,
		}
	}
	return Addr{
		addr: addr,
	}
}

// compute base32 address
func (addr Addr) Base32Addr() (b32 Base32Addr) {
	a := []byte(addr.addr)
	buf := make([]byte, i2pB64enc.DecodedLen(len(a)))
	n, err := i2pB64enc.Decode(buf, a)
	if err != nil {
		return
	}
	h := sha256.Sum256(buf[:n])
	copy(b32[:], h[:])
	return
}

// i2p destination hash
type Base32Addr [32]byte

// get string version
func (b32 Base32Addr) String() string {
	b32addr := make([]byte, 56)
	i2pB32enc.Encode(b32addr, b32[:])
	return string(b32addr[:52]) + ".b32.i2p"
}
