package i2p

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
)


// i2p network session
type Session interface {
	// get session name
	Name() string
	// open a new control socket
	// does handshaske
	OpenControlSocket() (net.Conn, error)

	// get our local address
	Addr() net.Addr
	
	// obtain new listener from this session
	Listen() (net.Listener, error)
	// lookup a name
	Lookup(name string) (net.Addr, error)
	// lookup an i2p address
	LookupI2P(name string) (I2PAddr, error)
	// dial
	Dial(n, a string) (net.Conn, error)
	// dial out to a remote destination
	DialI2P(a I2PAddr) (net.Conn, error)
	
	// open the session, generate keys, start up destination etc
	Open() error
}

type samSession struct {
	addr string
	minversion string
	maxversion string
	name string
	keys *Keyfile
	mtx sync.RWMutex
	// control connection
	c net.Conn
}

func (s *samSession) Name() string {
	return s.name
}

func (s *samSession) Addr() net.Addr {
	return s.keys.Addr()
}

func (s *samSession) OpenControlSocket() (n net.Conn, err error) {
	n, err = net.Dial("tcp", s.addr)
	if err == nil {
		_, err = fmt.Fprintf(n, "HELLO VERSION MIN=%s MAX=%s\n", s.minversion, s.maxversion)
		r := bufio.NewReader(n)
		var line string
		line, err = r.ReadString(10)
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
	var nc net.Conn
	nc, err = s.OpenControlSocket()
	if err == nil {
		// send connect
		_, err = fmt.Fprintf(nc, "STREAM CONNECT ID=%s DESTINATION=%s SILENT=false\n", s.Name(), addr.String())
		
		r := bufio.NewReader(nc)
		var line string
		// read reply
		line, err = r.ReadString(10)
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
					c = &I2PConn{
						c: nc,
						laddr: s.keys.Addr(),
						raddr: addr,
					}
					break
				}
				err = errors.New(line)
				nc.Close()
			}
		}
	}
	return
}

func (s *samSession) Dial(n, a string) (c net.Conn, err error) {
	if n == "i2p" {
		var addr I2PAddr
		addr, err = s.LookupI2P(a)
		if err == nil {
			c, err = s.DialI2P(addr)
		}
	} else {
		err = errors.New("cannot dial out to "+a+" network, not supported")
	}
	return
}

func (s *samSession) LookupI2P(name string) (a I2PAddr, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.c == nil {
		// no session socket
		err = errors.New("session not open")
		return
	}
	_, err = fmt.Fprintf(s.c, "NAMING LOOKUP NAME=%s\n", name)
	r := bufio.NewReader(s.c)
	var line string
	line, err = r.ReadString(10)
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
			if txt == fmt.Sprintf("NAME=%s", name) {
				continue
			}
			if strings.HasPrefix(txt, "VALUE=") {
				// we got it
				a = I2PAddr(txt[6:])
				return
			}
			err = errors.New(line)
		}
	}
	return
}

func (s *samSession) Lookup(name string) (a net.Addr, err error) {
	a, err = s.LookupI2P(name)
	return
}

func (s *samSession) Listen() (l net.Listener, err error) {
	// try opening if this session isn't already open
	if s.c == nil {
		err = s.Open()
	}
	if err == nil {
		// send session create
		_, err = fmt.Fprintf(s.c, "SESSION CREATE STYLE=STREAM ID=%s DESTINATION=%s\n", s.Name(), s.keys.privkey)
		if err == nil {
			// read response line
			r := bufio.NewReader(s.c)
			var line string
			line, err = r.ReadString(10)
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
						break
					}
					err = errors.New(line)
				}
				
				if err == nil {
					// do name lookup for ourself
					var us I2PAddr
					us, err = s.LookupI2P("ME")
					if err == nil {
						// we good
						l = &i2pListener{
							session: s,
							laddr: us,
						}
					}
				}
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
	return
}

// create a new i2p session
func NewSession(name, addr, keyfile string) Session {
	return &samSession{
		addr: addr,
		minversion: "3.0",
		maxversion: "3.0",
		keys: NewKeyfile(keyfile),
	}
}
