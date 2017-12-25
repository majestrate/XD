package common

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/url"
	"xd/lib/log"
	"xd/lib/network"
	"xd/lib/version"
)

// PeerID is a buffer for bittorrent peerid
type PeerID [20]byte

// Bytes gets buffer as byteslice
func (id PeerID) Bytes() []byte {
	return id[:]
}

// GeneratePeerID generates a new peer id for XD
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

// Peer provides info for a bittorrent swarm peer
type Peer struct {
	Compact []byte `bencode:"-"`
	IP      string `bencode:"ip"`
	Port    int    `bencode:"port"`
	ID      string `bencode:"id"`
}

func (p *Peer) PeerID() (id PeerID) {
	if len(p.ID) == 20 {
		copy(id[:], p.ID[:])
	}
	return
}

// Resolve resolves network address of peer
func (p *Peer) Resolve(n network.Network) (a net.Addr, err error) {
	log.Debugf("resolve %s", p)
	if len(p.IP) > 0 {
		a, err = n.Lookup(p.IP, fmt.Sprintf("%d", p.Port))
	} else {
		// try compact
		a, err = n.CompactToAddr(p.Compact[:], p.Port)
	}
	return
}
