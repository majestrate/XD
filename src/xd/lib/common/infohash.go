package common

import (
	"encoding/hex"
	"errors"
)

// ErrBadInfoHashLen is error indicating that the infohash is a bad size
var ErrBadInfoHashLen = errors.New("bad infohash length")

// Infohash is a hash digest of unspecified length compatable with both v1 and v2 torrents
type Infohash interface {
	// get full mutable byteslice
	Bytes() []byte
	// get hex representation of byteslice
	Hex() string
	// convert to v1 infohash
	ToV1() InfohashV1
}

// Infohash buffer for v2 bittorrent
type InfohashV2 [32]byte

// convert to v1 infohash via truncation
func (ihv2 InfohashV2) ToV1() (ihv1 InfohashV1) {
	copy(ihv1[:], ihv2[:])
	return
}

// get mutable byteslice
func (ihv2 InfohashV2) Bytes() []byte {
	return ihv2[:]
}

// hex representation of v2 infohash
func (ihv2 InfohashV2) Hex() string {
	return hex.EncodeToString(ihv2.Bytes())
}

// Infohash is a bittorrent infohash buffer
type InfohashV1 [20]byte

func (ih InfohashV1) ToV1() (ihv1 InfohashV1) {
	copy(ihv1[:], ih[:])
	return
}

// Hex gets hex representation of infohash
func (ih InfohashV1) Hex() string {
	return hex.EncodeToString(ih.Bytes())
}

// DecodeInfohash decode infohash from buffer
func DecodeInfohash(hexstr string) (ih Infohash, err error) {
	var ih1 InfohashV1
	var ih2 InfohashV2
	var dec []byte
	dec, err = hex.DecodeString(hexstr)
	if len(dec) == 32 {
		copy(ih2[:], dec[:])
		ih = ih2
	} else if len(dec) == 20 {
		copy(ih1[:], dec[:])
		ih = ih1
	} else {
		err = ErrBadInfoHashLen
	}
	return
}

// DecodeInfohashV2 decodes v2 infohash buffer from hex string
func DecodeInfohashV2(hexstr string) (ih InfohashV2, err error) {
	var dec []byte
	dec, err = hex.DecodeString(hexstr)
	if len(dec) == 32 {
		copy(ih[:], dec[:])
	} else {
		err = ErrBadInfoHashLen
	}
	return
}

// DecodeInfohash decodes infohash buffer from hex string
func DecodeInfohashV1(hexstr string) (ih InfohashV1, err error) {
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
func (ih InfohashV1) Bytes() []byte {
	return ih[:]
}
