package common

import (
	"encoding/hex"
)

// a bittorrent infohash
type Infohash [20]byte

// get hex representation
func (ih Infohash) Hex() string {
	return hex.EncodeToString(ih.Bytes())
}

// get underlying byteslice
func (ih Infohash) Bytes() []byte {
	return ih[:]
}
