package common

import (
	"bytes"
	"encoding/hex"
	"errors"
)

var ErrBadMagnetURI = errors.New("bad magnet URI")

// ErrBadInfoHashLen is error indicating that the infohash is a bad size
var ErrBadInfoHashLen = errors.New("bad infohash length")

// Infohash is a bittorrent infohash buffer
type Infohash [20]byte

func (ih Infohash) Equal(other Infohash) bool {
	return bytes.Equal(ih.Bytes(), other.Bytes())
}

// Hex gets hex representation of infohash
func (ih Infohash) Hex() string {
	return hex.EncodeToString(ih.Bytes())
}

// DecodeInfohash decodes infohash buffer from hex string
func DecodeInfohash(hexstr string) (ih Infohash, err error) {
	var dec []byte
	dec, err = hex.DecodeString(hexstr)
	if len(dec) == 20 {
		copy(ih[:], dec[:])
	} else {
		err = ErrBadInfoHashLen
	}
	return
}

// Bytes gets underlying byteslice of infohash buffer
func (ih Infohash) Bytes() []byte {
	return ih[:]
}
