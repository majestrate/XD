package common

import (
	"crypto/rand"
	"io"
	"net/url"
	"xd/lib/i2p"
	"xd/lib/version"
)

type PeerID [20]byte

func (id PeerID) Bytes() []byte {
	return id[:]
}

// generate a new peer id
func (id PeerID) Generate() {
	io.ReadFull(rand.Reader, id[:])
	copy(id[:], []byte(version.Version))
}

// encode to string
func (id PeerID) String() string {
	return url.QueryEscape(string(id.Bytes()))
}

// swarm peer
type Peer struct {
	Compact i2p.Base32Addr `bencode:"-"`
	IP string `bencode:"ip"`
	Port int `bencode:"port"`
	ID PeerID `bencode:"id"`
}
