package network

import (
	"net"
)

// a network session
type Network interface {
	Dial(n, a string) (net.Conn, error)
	Accept() (net.Conn, error)
	Open() error
	Close() error
	Addr() net.Addr
	Lookup(name, port string) (net.Addr, error)
	CompactToAddr(compact []byte, port int) (net.Addr, error)
	AddrToCompact(addr string) []byte
	B32Addr() string
	SaveKey(fname string) error
}
