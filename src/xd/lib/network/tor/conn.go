package tor

import (
	"net"
	"time"
)

type OnionConn struct {
	laddr *OnionAddr
	raddr *OnionAddr
	conn  net.Conn
}

func (c *OnionConn) LocalAddr() net.Addr {
	return c.laddr
}

func (c *OnionConn) RemoteAddr() net.Addr {
	return c.raddr
}

func (c *OnionConn) Close() error {
	return c.conn.Close()
}

func (c *OnionConn) Write(d []byte) (int, error) {
	return c.conn.Write(d)
}

func (c *OnionConn) Read(d []byte) (int, error) {
	return c.conn.Read(d)
}

func (c *OnionConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *OnionConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *OnionConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
