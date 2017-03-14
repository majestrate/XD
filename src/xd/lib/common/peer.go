package common

import (
	"crypto/rand"
	"io"
	"net"
	"net/url"
	"strings"
	"xd/lib/i2p"
	"xd/lib/network"
	"xd/lib/version"
)

type PeerID [20]byte

func (id PeerID) Bytes() []byte {
	return id[:]
}

// generate a new peer id
func GeneratePeerID() (id PeerID) {
	io.ReadFull(rand.Reader, id[:])
	id[0] = '-'
	v := version.Name + version.Major + version.Minor + version.Patch + "0-"
	copy(id[1:], []byte(v[:]))
	return
}

// encode to string
func (id PeerID) String() string {
	return url.QueryEscape(string(id.Bytes()))
}

// swarm peer
type Peer struct {
	Compact i2p.Base32Addr `bencode:"-"`
	IP      string         `bencode:"ip"`
	Port    int            `bencode:"port"`
	ID      PeerID         `bencode:"id"`
}

// resolve network address
func (p *Peer) Resolve(n network.Network) (a net.Addr, err error) {
	if len(p.IP) > 0 {
		// prefer ip
		parts := strings.Split(p.IP, ".i2p")
		a = i2p.I2PAddr(parts[0])
	} else {
		// try compact
		a, err = n.Lookup(p.Compact.String())
	}
	return
}
