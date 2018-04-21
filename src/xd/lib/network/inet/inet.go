package inet

import (
	"net"
)

type Session struct {
	LocalAddr net.Addr
	serv      net.Listener
	packet    net.PacketConn
}

func (s *Session) Dial(n, a string) (net.Conn, error) {
	return net.Dial(n, a)
}

func (s *Session) Accept() (net.Conn, error) {
	return s.serv.Accept()
}

func (s *Session) Open() (err error) {
	s.serv, err = net.Listen(s.LocalAddr.Network(), s.LocalAddr.String())
	return
}

func (s *Session) ReadFrom(d []byte) (n int, from net.Addr, err error) {
	return
}

func (s *Session) WriteTo(d []byte, to net.Addr) (n int, err error) {
	return
}

func (s *Session) Close() error {
	return s.serv.Close()
}

func (s *Session) Addr() net.Addr {
	return s.LocalAddr
}

func (s *Session) Lookup(name, port string) (addr net.Addr, err error) {
	var ips []net.IP
	ips, err = net.LookupIP(name)
	if err == nil {
		for _, ip := range ips {
			tcpaddr := &net.TCPAddr{
				IP: ip,
			}
			tcpaddr.Port, err = net.LookupPort(tcpaddr.Network(), port)
			if err == nil {
				addr = tcpaddr
				return
			}
		}
	}
	return
}
