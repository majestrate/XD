package common

import (
	"encoding/hex"
	"errors"
)

var ErrBadInfoHashLen = errors.New("bad infohash length")

// a bittorrent infohash
type Infohash [20]byte

// get hex representation
func (ih Infohash) Hex() string {
	return hex.EncodeToString(ih.Bytes())
}

func (ih Infohash) Decode(hexstr string) (err error) {
	var dec []byte
	dec, err = hex.DecodeString(hexstr)
	if len(dec) == 20 {
		copy(ih[:], dec[:])
	} else {
		err = ErrBadInfoHashLen
	}
	return
}

// get underlying byteslice
func (ih Infohash) Bytes() []byte {
	return ih[:]
}
