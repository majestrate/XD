package dht

import "github.com/zeebo/bencode"

type Serialize interface {
	bencode.Marshaler
	bencode.Unmarshaler
}
