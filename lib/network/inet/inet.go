package inet

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

const DefaultIfName = "lokitun0"
const DefaultPort = "0"

type Session struct {
	localIP   net.IP
	localAddr string
	name      string
	port      string
	serv      net.Listener
	packet    net.PacketConn
	resolver  net.Resolver
}

func NewSession(ifname, port, dns string) (s *Session, err error) {
	var netif *net.Interface
	netif, err = net.InterfaceByName(ifname)
	if err != nil {
		return
	}
	var ifaddrs []net.Addr
	ifaddrs, err = netif.Addrs()
	if err != nil {
		return
	}
	if len(ifaddrs) == 0 {
		err = fmt.Errorf("%s has no addresses? duh fug yo...", ifname)
		return
	}
	var localIP net.IP
	localIP, _, err = net.ParseCIDR(ifaddrs[0].String())
	if err != nil {
		return
	}
	ss := &Session{
		port:      port,
		localIP:   localIP,
		localAddr: net.JoinHostPort(localIP.String(), port),
		resolver: net.Resolver{
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "udp", dns)
			},
		},
	}
	var names []string
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	names, err = ss.resolver.LookupAddr(ctx, ss.localIP.String())
	if err != nil {
		return
	}
	if len(names) == 0 {
		err = fmt.Errorf("we have no rdns record for %s", ifname)
		return
	}
	ss.name = strings.TrimSuffix(names[0], ".")
	s = ss
	return
}

func (s *Session) LocalName() string {
	return s.name
}

func (s *Session) Dial(_, a string) (net.Conn, error) {
	h, p, err := net.SplitHostPort(a)
	if err != nil {
		return nil, err
	}
	raddr, err := s.lookupTCP(h, p)
	if err != nil {
		return nil, err
	}
	laddr, err := net.ResolveTCPAddr("tcp4", s.localAddr)
	if err != nil {
		return nil, err
	}
	c, err := net.DialTCP("tcp4", laddr, raddr)
	if err != nil {
		return nil, err
	}
	return s.wrapConn(c)
}

func (s *Session) wrapConn(c net.Conn) (*Conn, error) {
	raddr := c.RemoteAddr()
	h, port, err := net.SplitHostPort(raddr.String())
	if err != nil {
		c.Close()
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	names, err := s.resolver.LookupAddr(ctx, h)
	if err != nil {
		c.Close()
		return nil, err
	}
	return &Conn{
		c: c,
		laddr: &Addr{
			name: s.name,
			port: s.port,
		},
		raddr: &Addr{
			name: strings.TrimSuffix(names[0], "."),
			port: port,
		},
	}, nil
}

type Listener struct {
	l     net.Listener
	laddr *Addr
}

func (l *Listener) Addr() net.Addr {
	return l.laddr
}

func (l *Listener) Close() error {
	return l.l.Close()
}

func (l *Listener) Accept() (net.Conn, error) {
	return l.l.Accept()
}

func NewAddr(n, p string) *Addr {
	return &Addr{
		name: n,
		port: p,
	}
}

type Addr struct {
	name string
	port string
}

func (a *Addr) Network() string {
	return "tcp"
}

func (a *Addr) String() string {
	return net.JoinHostPort(a.name, a.port)
}

type Conn struct {
	c     net.Conn
	laddr *Addr
	raddr *Addr
}

// implements net.Conn
func (c *Conn) Read(d []byte) (n int, err error) {
	n, err = c.c.Read(d)
	return
}

// implements net.Conn
func (c *Conn) Write(d []byte) (n int, err error) {
	n, err = c.c.Write(d)
	return
}

// implements net.Conn
func (c *Conn) Close() error {
	return c.c.Close()
}

// implements net.Conn
func (c *Conn) LocalAddr() net.Addr {
	return c.laddr
}

// implements net.Conn
func (c *Conn) RemoteAddr() net.Addr {
	return c.raddr
}

// implements net.Conn
func (c *Conn) SetDeadline(t time.Time) error {
	return c.c.SetDeadline(t)
}

// implements net.Conn
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.c.SetReadDeadline(t)
}

// implements net.Conn
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.c.SetWriteDeadline(t)
}

func (s *Session) Accept() (net.Conn, error) {
	c, err := s.serv.Accept()
	if err != nil {
		return nil, err
	}
	return s.wrapConn(c)
}

func (s *Session) Open() error {
	l, err := net.Listen("tcp", s.localAddr)
	if err != nil {
		return err
	}
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return err
	}
	s.serv = &Listener{
		l: l,
		laddr: &Addr{
			name: s.name,
			port: port,
		},
	}
	return nil
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
	if s.serv == nil {
		return nil
	}
	return s.serv.Addr()
}

func (s *Session) Lookup(name, port string) (addr net.Addr, err error) {
	return s.lookupTCP(name, port)
}

func (s *Session) lookupTCP(name, port string) (addr *net.TCPAddr, err error) {
	var ips []net.IPAddr
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	ips, err = s.resolver.LookupIPAddr(ctx, name)
	if err == nil {
		for _, ip := range ips {
			tcpaddr := &net.TCPAddr{
				IP: ip.IP,
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
