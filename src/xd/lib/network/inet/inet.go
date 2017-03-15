package inet

import (
	"net"
)

type Session struct {
	LocalAddr net.Addr
	serv      net.Listener
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

func (s *Session) Close() error {
	return s.serv.Close()
}

func (s *Session) Addr() net.Addr {
	return s.LocalAddr
}

func (s *Session) Lookup(name string, port int) (addr net.Addr, err error) {
	var ips []net.IP
	ips, err = net.LookupIP(name)
	if err == nil {
		addr = &net.TCPAddr{
			IP:   ips[0],
			Port: port,
		}
	}
	return
}
