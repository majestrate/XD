package gnutella

import (
	"net"
	"net/textproto"
)

type Conn struct {
	c   net.Conn
	tpc *textproto.Conn
}

func (c *Conn) Handshake(reject bool) (err error) {

	if reject {
		return c.tpc.PrintfLine("GNUTELLA/0.6 503 Rejected")
	}
	return err
}

func (c *Conn) Close() error {
	return c.c.Close()
}

func NewConn(c net.Conn) *Conn {
	return &Conn{
		c:   c,
		tpc: textproto.NewConn(c),
	}
}
