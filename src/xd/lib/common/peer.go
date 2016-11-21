package common

import (
	"net/url"
	"xd/lib/i2p"
)

type PeerID [20]byte

func (id PeerID) Bytes() []byte {
	return id[:]
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
