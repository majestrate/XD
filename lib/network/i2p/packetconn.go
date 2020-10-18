package i2p

import (
	"bytes"
	"net"
	"time"
)

// tcp/i2p connection
// implements net.Conn
type I2PPacketConn struct {
	// underlying connection
	c net.PacketConn
	// our local address
	laddr Addr
	// remote sam addr
	samaddr net.Addr
	// sam version
	version string
}

// implements net.PacketConn
func (c *I2PPacketConn) ReadFrom(d []byte) (n int, from net.Addr, err error) {
	var buff [65336]byte
	for err == nil {
		n, from, err = c.c.ReadFrom(buff[:])
		if err == nil {
			if from.String() != c.samaddr.String() {
				// drop silent because source missmatch
				continue
			}
			idx := bytes.IndexByte(buff[:n], 10)
			if idx <= 0 {
				// drop silent because invalid format
				continue
			}
			parts := bytes.SplitN(buff[:idx-1], []byte{' '}, 2)
			if len(parts) < 2 {
				// drop silent because invalid format
				continue
			}
			parts = bytes.Split(parts[1], []byte{' '})

			from = I2PAddr(string(parts[len(parts)-1]))
			data := buff[idx+1 : n]
			n -= 1 + idx
			if len(d) < n {
				// drop silent because too big for caller
				continue
			}

			copy(d, data)
			break
		}
	}
	return
}

// implements net.PacketConn
func (c *I2PPacketConn) WriteTo(d []byte, to net.Addr) (n int, err error) {
	tostr := c.version + " " + to.String()
	tolen := len(tostr)
	buff := make([]byte, len(d)+tolen+1)
	copy(buff, tostr)
	buff[tolen] = '\n'
	copy(buff[:tolen+1], d)
	n, err = c.c.WriteTo(buff, c.samaddr)
	if err == nil {
		n = len(d)
	}
	return
}

// implements net.PacketConn
func (c *I2PPacketConn) Close() error {
	if c.c == nil {
		return nil
	}
	return c.c.Close()
}

// implements net.PacketConn
func (c *I2PPacketConn) LocalAddr() net.Addr {
	return c.laddr
}

// implements net.PacketConn
func (c *I2PPacketConn) SetDeadline(t time.Time) error {
	return c.c.SetDeadline(t)
}

// implements net.PacketConn
func (c *I2PPacketConn) SetReadDeadline(t time.Time) error {
	return c.c.SetReadDeadline(t)
}

// implements net.PacketConn
func (c *I2PPacketConn) SetWriteDeadline(t time.Time) error {
	return c.c.SetWriteDeadline(t)
}
