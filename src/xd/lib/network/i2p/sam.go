package i2p

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type lookupResp struct {
	addr I2PAddr
	err  error
}

type lookupReq struct {
	name      string
	replyChnl chan lookupResp
}

type samSession struct {
	style      string
	addr       string
	minversion string
	maxversion string
	name       string
	keys       *Keyfile
	opts       map[string]string
	c          net.Conn
	readbuf    [1]byte
	lookup     chan *lookupReq
	pktconn    I2PPacketConn
}

func (s *samSession) ReadFrom(d []byte) (n int, from net.Addr, err error) {
	if s.style != "DATAGRAM" {
		err = errors.New("not supported")
		return
	}
	n, from, err = s.pktconn.ReadFrom(d)
	return
}

func (s *samSession) WriteTo(d []byte, to net.Addr) (n int, err error) {
	if s.style != "DATAGRAM" {
		err = errors.New("not supported")
		return
	}
	n, err = s.pktconn.WriteTo(d, to)
	return
}

func (s *samSession) Close() error {
	if s.c == nil {
		return nil
	}
	s.lookup <- nil
	err := s.c.Close()
	if s.style == "DATAGRAM" {
		s.pktconn.Close()
	}
	s.c = nil
	return err
}

func (s *samSession) B32Addr() string {
	return s.keys.Addr().Base32Addr().String()
}

func (s *samSession) Name() string {
	return s.name
}

func (s *samSession) Addr() net.Addr {
	return s.keys.Addr()
}

func (s *samSession) OpenControlSocket() (n net.Conn, err error) {
	readbuf := make([]byte, 1)
	n, err = net.Dial("tcp", s.addr)
	if err == nil {
		// make the connection never time out
		err = n.(*net.TCPConn).SetKeepAlive(true)
		if err == nil {
			// send keepalive every 5 seconds
			err = n.(*net.TCPConn).SetKeepAlivePeriod(time.Second * 5)
		}
		if err != nil {
			err = nil
		}
		_, err = fmt.Fprintf(n, "HELLO VERSION MIN=%s MAX=%s\n", s.minversion, s.maxversion)
		var line string
		line, err = readLine(n, readbuf)
		if err == nil {
			sc := bufio.NewScanner(strings.NewReader(line))
			sc.Split(bufio.ScanWords)
			for sc.Scan() {
				text := strings.ToUpper(sc.Text())
				if text == "HELLO" {
					continue
				}
				if text == "REPLY" {
					continue
				}
				if text == "RESULT=OK" {
					// we good
					return
				}
				err = errors.New(line)
			}
		}
		n.Close()
	}
	return
}

func (s *samSession) DialI2P(addr I2PAddr) (c net.Conn, err error) {
	readbuf := make([]byte, 1)
	var nc net.Conn
	nc, err = s.OpenControlSocket()
	if err == nil {
		// send connect
		_, err = fmt.Fprintf(nc, "STREAM CONNECT ID=%s DESTINATION=%s SILENT=false\n", s.Name(), addr.String())
		var line string
		// read reply
		line, err = readLine(nc, readbuf)
		if err == nil {
			// parse reply
			sc := bufio.NewScanner(strings.NewReader(line))
			sc.Split(bufio.ScanWords)
			for sc.Scan() {
				txt := sc.Text()
				upper := strings.ToUpper(txt)
				if upper == "STREAM" {
					continue
				}
				if upper == "STATUS" {
					continue
				}
				if upper == "RESULT=OK" {
					// we are connected
					err = nc.(*net.TCPConn).SetKeepAlive(false)
					c = &I2PConn{
						c:     nc,
						laddr: s.keys.Addr(),
						raddr: addr,
					}
					return
				}
				err = errors.New(line)
				nc.Close()
			}
		}
	}
	return
}

func (s *samSession) Dial(n, a string) (c net.Conn, err error) {
	var addr I2PAddr
	addr, err = s.LookupI2P(a)
	if err == nil {
		c, err = s.DialI2P(addr)
	}
	return
}

func (s *samSession) LookupI2P(name string) (a I2PAddr, err error) {
	var n string
	n, _, err = net.SplitHostPort(name)
	if err == nil {
		name = n
	}
	req := lookupReq{
		replyChnl: make(chan lookupResp),
		name:      name,
	}
	s.lookup <- &req
	repl := <-req.replyChnl
	a, err = repl.addr, repl.err
	return
}

func (s *samSession) runLookups() {
	var err error
	for err == nil {
		var resp lookupResp
		req := <-s.lookup
		if req == nil {
			return
		}
		c := s.c
		if c == nil {
			return
		}
		_, err = fmt.Fprintf(c, "NAMING LOOKUP NAME=%s\n", req.name)
		var line string
		line, err = readLine(c, s.readbuf[:])
		if err == nil {
			// okay
			sc := bufio.NewScanner(strings.NewReader(line))
			sc.Split(bufio.ScanWords)
			for sc.Scan() {
				txt := sc.Text()
				upper := strings.ToUpper(txt)
				if upper == "NAMING" {
					continue
				}
				if upper == "REPLY" {
					continue
				}
				if upper == "RESULT=OK" {
					continue
				}
				if strings.HasPrefix(upper, "NAME=") {
					continue
				}
				if strings.HasPrefix(txt, "VALUE=") {
					// we got it
					resp.addr = I2PAddr(txt[6:])
					break
				}
				resp.err = errors.New(line)
				break
			}
		} else {
			resp.err = err
		}
		req.replyChnl <- resp
	}
	return
}

func (s *samSession) Lookup(name, port string) (a net.Addr, err error) {
	a, err = s.LookupI2P(name)
	return
}

func (s *samSession) udpAddr() (string, error) {
	host, port, err := net.SplitHostPort(s.addr)
	if err != nil {
		return "", err
	}
	var addrs []net.Addr
	addrs, err = net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	var srcaddr *net.IPAddr
	srcaddr, err = net.ResolveIPAddr("ip", host)
	if err != nil {
		return "", err
	}
	for idx := range addrs {
		var ipnet *net.IPNet
		var srcip net.IP
		srcip, ipnet, err = net.ParseCIDR(addrs[idx].String())
		if err != nil {
			return "", err
		}
		if ipnet.Contains(srcaddr.IP) {
			var pint int
			pint, err = net.LookupPort("udp", port)
			if err != nil {
				return "", err
			}
			return net.JoinHostPort(srcip.String(), fmt.Sprintf("%d", pint-1)), nil
		}
	}
	return "", errors.New("unroutable address: " + host)
}

func (s *samSession) createSession() (err error) {
	// try opening if this session isn't already open
	optsstr := " inbound.name=XD"
	if s.opts != nil {
		for k, v := range s.opts {
			optsstr += fmt.Sprintf(" %s=%s", k, v)
		}
	}
	if s.style == "DATAGRAM" {
		var addr string
		addr, err = s.udpAddr()
		if err != nil {
			return
		}
		s.pktconn.c, err = net.ListenPacket("udp", addr)
		if err != nil {
			return
		}
		addr = s.pktconn.c.LocalAddr().String()
		var host, port string
		host, port, err = net.SplitHostPort(addr)
		if err != nil {
			s.pktconn.c.Close()
			return err
		}
		optsstr += fmt.Sprintf(" HOST=%s PORT=%s", host, port)
	}

	_, err = fmt.Fprintf(s.c, "SESSION CREATE STYLE=%s ID=%s SIGNATURE_TYPE=%d DESTINATION=%s%s\n", s.style, s.Name(), SigType, s.keys.privkey, optsstr)
	if err == nil {
		// read response line
		var line string
		line, err = readLine(s.c, s.readbuf[:])
		if err == nil {
			// parse response line
			sc := bufio.NewScanner(strings.NewReader(line))
			sc.Split(bufio.ScanWords)
			for sc.Scan() {
				text := sc.Text()
				upper := strings.ToUpper(text)
				if upper == "SESSION" {
					continue
				}
				if upper == "STATUS" {
					continue
				}
				if upper == "RESULT=OK" {
					// we good
					return
				}
				err = errors.New(line)
			}
		}
	}
	return
}

func (s *samSession) Open() (err error) {
	s.c, err = s.OpenControlSocket()
	if err == nil {
		err = s.keys.ensure(s.c)
	}
	if err == nil {
		err = s.createSession()
		if err == nil {
			go s.runLookups()
			var a I2PAddr
			a, err = s.LookupI2P("ME")
			if err == nil {
				s.keys.pubkey = a.String()
				s.pktconn.laddr = a
			}
		}
	}
	if err != nil {
		s.Close()
	}
	return
}

func (s *samSession) Accept() (c net.Conn, err error) {
	l := &i2pListener{
		session: s,
		laddr:   I2PAddr(s.keys.pubkey),
	}
	c, err = l.Accept()
	return
}
