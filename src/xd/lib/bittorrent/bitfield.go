package bittorrent

import (
	"github.com/zeebo/bencode"
	"io"
)

type Bitfield struct {
	// length in bits
	Length int `bencode:"bits"`
	// bitfield data
	Data []byte `bencode:"bitfield"`
}

// create new bitfield
func NewBitfield(l int, d []byte) *Bitfield {
	b := make([]byte, len(d))
	copy(b, d)
	return &Bitfield{
		Length: l,
		Data: b,
	}
}

// for fs storage
func (bf *Bitfield) BEncode(w io.Writer) (err error) {
	enc := bencode.NewEncoder(w)
	err = enc.Encode(bf)
	return
}

// for fs storage
func (bf *Bitfield) BDecode(r io.Reader) (err error) {
	dec := bencode.NewDecoder(r)
	err = dec.Decode(bf)
	return
}

// serialize to wire message
func (bf *Bitfield) ToWireMessage() *WireMessage {
	return NewWireMessage(5, bf.Data[:])
}

func (bf *Bitfield) Set(p int) {
	
}

func (bf *Bitfield) Has(p int) bool {
	if p < bf.Length {
		return bf.Data[p>>3]&(1<<(7-uint(p)&7)) != 0
	}
	return false
}
