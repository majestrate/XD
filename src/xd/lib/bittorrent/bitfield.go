package bittorrent

import (
	"bytes"
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
	if d == nil {
		d = make([]byte, (l/8)+1)
	}
	b := make([]byte, len(d))
	copy(b, d)
	return &Bitfield{
		Length: l,
		Data:   b,
	}
}

// get as inverted
func (bf *Bitfield) Inverted() (i *Bitfield) {
	i = NewBitfield(bf.Length, nil)
	bit := 0
	for bit < bf.Length {
		if !bf.Has(bit) {
			i.Set(bit)
		}
		bit++
	}
	return
}

func (bf *Bitfield) Equals(other *Bitfield) bool {
	return bytes.Equal(bf.Data, other.Data)
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
	return NewWireMessage(BitField, bf.Data[:])
}

func (bf *Bitfield) Set(p int) {
	if p < bf.Length {
		idx := p >> 3
		if idx < len(bf.Data) {
			bf.Data[idx] |= (1 << (7 - uint(p)&7))
		}
	}
}

// count how many bits are set
func (bf *Bitfield) CountSet() (sum int) {
	l := bf.Length
	for l > 0 {
		l--
		// TODO: make this less horrible
		if bf.Has(l) {
			sum++
		}
	}
	return
}

func (bf *Bitfield) Has(p int) bool {
	if p < bf.Length {
		idx := p >> 3
		if idx < len(bf.Data) {
			return bf.Data[idx]&(1<<(7-uint(p)&7)) != 0
		}
	}
	return false
}

func (bf *Bitfield) Completed() bool {
	l := bf.Length
	for l > 0 {
		l--
		// TODO: make this less horrible
		if !bf.Has(l) {
			return false
		}
	}
	return true
}
