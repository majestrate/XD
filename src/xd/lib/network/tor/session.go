package tor

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base32"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/yawning/bulb"
	"github.com/yawning/bulb/utils/pkcs1"
	"golang.org/x/net/proxy"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"
	"xd/lib/log"
)

var ErrNotFound = errors.New("host not found")
var ErrAcceptFailed = errors.New("acccept failed")
var ErrInternalFail = errors.New("internal failure")
var ErrBadDomain = errors.New("bad domain")
var ErrBadCert = errors.New("bad cert")

type evSubscription struct {
	chnl chan *bulb.Response
	name string
}

func (sub *evSubscription) Inform(r *bulb.Response) {
	sub.chnl <- r
}

type eventSub struct {
	subs   []evSubscription
	access sync.Mutex
}

func (ev *eventSub) Sub(chnl chan *bulb.Response, name string) {
	ev.access.Lock()
	ev.subs = append(ev.subs, evSubscription{chnl, name})
	ev.access.Unlock()
}

func (ev *eventSub) Inform(r *bulb.Response, name string) {
	ev.access.Lock()
	remove := make(map[int]bool)
	for idx, s := range ev.subs {
		if s.name == name {
			s.Inform(r)
			remove[idx] = true
		} else {
			remove[idx] = false
		}
	}
	old := ev.subs
	ev.subs = []evSubscription{}
	for idx := range old {
		if remove[idx] {
			continue
		} else {
			ev.subs = append(ev.subs, old[idx])
		}
	}
	ev.access.Unlock()
}

type Session struct {
	net        string
	addr       string
	keys       string
	passwd     string
	conn       *bulb.Conn
	l          net.Listener
	tlsConfig  tls.Config
	inbound    chan *OnionConn
	onionInfo  *bulb.OnionInfo
	ourCert    x509.Certificate
	subs       map[string]*eventSub
	nameCache  map[string]rsa.PublicKey
	nameAccess sync.Mutex
	port       int
}

func (s *Session) getNameCache(name string) (k rsa.PublicKey, ok bool) {
	s.nameAccess.Lock()
	k, ok = s.nameCache[name]
	s.nameAccess.Unlock()
	return
}

func (s *Session) putNameCache(name string, k rsa.PublicKey) {
	s.nameAccess.Lock()
	s.nameCache[name] = k
	s.nameAccess.Unlock()
}

func (s *Session) subscribe(ev, name string) chan *bulb.Response {
	chnl := make(chan *bulb.Response)
	sub, ok := s.subs[ev]
	if !ok {
		sub = new(eventSub)
		s.subs[ev] = sub
	}
	sub.Sub(chnl, name)
	return chnl
}

func (s *Session) Accept() (c net.Conn, err error) {
	c, err = s.l.Accept()
	return
}

func (s *Session) Addr() net.Addr {
	return &OnionAddr{
		k: s.publicKey(),
		p: s.port,
	}
}

func (s *Session) Lookup(name, port string) (net.Addr, error) {
	return s.LookupOnion(name, port)
}

func (s *Session) lookupConn() (*bulb.Conn, error) {
	c, err := bulb.Dial(s.net, s.addr)
	if c != nil {
		err = c.Authenticate(s.passwd)
		if err != nil {
			c.Close()
			c = nil
		}
	}
	return c, err
}

func (s *Session) runEvents() {
	r, _ := s.conn.Request("SETEVENTS HS_DESC_CONTENT")
	if r.IsOk() {
		var err error
		s.conn.StartAsyncReader()
		for err == nil {
			var ev *bulb.Response
			log.Debug("read next event")
			ev, err = s.conn.NextEvent()
			if err == nil {
				if len(ev.Data) > 0 {
					firstLine := ev.Data[0]
					parts := strings.Split(firstLine, " ")
					sub, ok := s.subs[parts[0]]
					if ok {
						sub.Inform(ev, parts[1])
					}
				}
			}
		}
	} else {
		log.Error("error setting events")
	}
}

func (s *Session) lookupInform(name string, chnl chan *OnionAddr) {
	ch := s.subscribe("HS_DESC_CONTENT", name)
	r := <-ch
	addr := new(OnionAddr)
	foundKey := false
	var buff bytes.Buffer
	var lines []string
	for _, data := range r.Data {
		lines = append(lines, strings.Split(data, "\n")...)
	}
	for _, line := range lines {
		if foundKey {
			io.WriteString(&buff, line)
			if strings.ToUpper(line) == "-----END RSA PUBLIC KEY-----" {
				block, err := pem.Decode(buff.Bytes())
				if block == nil {
					log.Errorf("error decoding pem: %s", err)
					chnl <- nil
				} else {
					_, _ = asn1.Unmarshal(block.Bytes, &addr.k)
					o := addr.Onion()
					s.putNameCache(o[:len(o)-6], addr.k)
					chnl <- addr
				}
			} else {
				buff.Write([]byte{10})
			}
		} else {
			log.Debugf("line: %s", line)
			if line == "permanent-key" {
				foundKey = true
			}
		}
	}
}

func (s *Session) LookupOnion(name, port string) (a *OnionAddr, err error) {
	name = strings.ToLower(name)
	log.Debugf("lookup: %s", name)
	if strings.HasSuffix(name, ".onion") {
		hs := name[:len(name)-6]
		k, ok := s.getNameCache(hs)
		if ok {
			a = &OnionAddr{
				k: k,
			}
			a.p, err = net.LookupPort("tcp", port)
			if err != nil {
				a = nil
			}
			return
		}
		var conn *bulb.Conn
		conn, err = s.lookupConn()
		if err == nil {
			var r *bulb.Response
			hs := name[:len(name)-6]
			r, err = conn.Request("HSFETCH %s", hs)
			if err == nil {
				if r.IsOk() {
					chnl := make(chan *OnionAddr)
					go s.lookupInform(hs, chnl)
					a = <-chnl
					if a == nil {
						err = ErrInternalFail
					}
				} else {
					err = ErrInternalFail
				}
			}
			conn.Close()
		}
	}
	return
}

func (s *Session) CompactToAddr(compact []byte, port int) (a net.Addr, err error) {
	hsaddr := strings.Trim(base32.HexEncoding.EncodeToString(compact), "=")
	a, err = s.Lookup(hsaddr+".onion", fmt.Sprintf("%d", port))
	return
}

func (s *Session) AddrToCompact(addr string) []byte {
	host, _, _ := net.SplitHostPort(addr)
	hs := host[:len(host)-6]
	b, _ := base32.HexEncoding.DecodeString(hs)
	return b
}

func (s *Session) publicKey() rsa.PublicKey {
	return s.onionInfo.PrivateKey.(*rsa.PrivateKey).PublicKey
}

func (s *Session) B32Addr() string {
	addr := s.publicKey()
	id, _ := pkcs1.OnionAddr(&addr)
	return id
}

func (s *Session) verifyPeerCert(certs [][]byte, _ [][]*x509.Certificate) (err error) {
	for _, certRaw := range certs {
		var cert *x509.Certificate
		cert, err = x509.ParseCertificate(certRaw)
		if err != nil {
			return
		}
		if len(cert.DNSNames) != 1 {
			continue
		}
		pool := x509.NewCertPool()
		pool.AddCert(cert)
		name := cert.DNSNames[0]
		_, err = cert.Verify(x509.VerifyOptions{
			DNSName:   name,
			Roots:     pool,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		})
		if err == nil {
			err = s.HostExists(name)
			return
		}
	}
	if err == nil {
		err = ErrBadCert
	}
	return
}

func (s *Session) HostExists(onion string) (err error) {
	var addr *OnionAddr
	addr, err = s.LookupOnion(onion, "0")
	if addr == nil {
		if addr.Onion() != onion {
			err = ErrNotFound
		}
	}
	return
}

func (s *Session) setupTLS() (err error) {
	name := s.B32Addr()
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	var num *big.Int
	num, err = rand.Int(rand.Reader, serialNumberLimit)
	s.ourCert = x509.Certificate{
		SerialNumber: num,
		Subject: pkix.Name{
			Organization: []string{"INTERNET"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(900000 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:     true,
		DNSNames: []string{name},
	}
	var certBytes []byte
	pubkey := s.publicKey()
	certBytes, err = x509.CreateCertificate(rand.Reader, &s.ourCert, &s.ourCert, &pubkey, s.onionInfo.PrivateKey)
	if err == nil {
		log.Debug("setting tls")
		var cert tls.Certificate
		cert.PrivateKey = s.onionInfo.PrivateKey
		cert.Certificate = append(cert.Certificate, certBytes)
		s.tlsConfig.Certificates = []tls.Certificate{cert}
		s.tlsConfig.CipherSuites = []uint16{tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305}
		s.tlsConfig.ClientAuth = tls.RequireAnyClientCert
		s.tlsConfig.ServerName = name
		s.tlsConfig.VerifyPeerCertificate = s.verifyPeerCert
		s.tlsConfig.InsecureSkipVerify = true
	}
	return
}

func (s *Session) Open() (err error) {
	if s.conn == nil {
		s.conn, err = bulb.Dial(s.net, s.addr)
		log.Debugf("Dial to %s %s", s.net, s.addr)
		if err == nil {
			var k *rsa.PrivateKey
			k, err = rsa.GenerateKey(rand.Reader, 1024)
			log.Debug("create rsa")
			if err == nil {
				err = s.conn.Authenticate(s.passwd)
				if err == nil {
					s.l, s.onionInfo, err = s.conn.NewListener(&bulb.NewOnionConfig{
						PrivateKey: k,
						DiscardPK:  true,
					}, uint16(s.port))
					if err == nil {
						s.onionInfo.PrivateKey = k
						log.Debug("made onion")
						err = s.setupTLS()
						if err == nil {
							go s.runEvents()
							log.Debug("tls set up")
							s.l = tls.NewListener(s.l, s.tlsConfig.Clone())
						} else {
							s.Close()
						}
					}
				}
			}
		}
	}
	return
}

func (s *Session) Close() (err error) {
	if s.l != nil {
		s.l.Close()
		s.l = nil
	}
	if s.conn != nil {
		err = s.conn.Close()
		s.conn = nil
	}
	return
}

func (s *Session) Dial(n, a string) (c net.Conn, err error) {
	var d proxy.Dialer
	d, err = s.conn.Dialer(nil)
	if err == nil {
		c, err = d.Dial(n, a)
		if err == nil {
			c = tls.Client(c, s.tlsConfig.Clone())
		}
	}
	return
}

func (s *Session) SaveKey(fname string) (err error) {
	err = ioutil.WriteFile(fname, x509.MarshalPKCS1PrivateKey(s.onionInfo.PrivateKey.(*rsa.PrivateKey)), 0600)
	return
}
