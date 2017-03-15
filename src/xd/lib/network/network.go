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
	Lookup(name string, port int) (net.Addr, error)
}
