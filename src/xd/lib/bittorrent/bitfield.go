package bittorrent

import (
	"bytes"
	"io"
	"xd/lib/common"

	"github.com/zeebo/bencode"
)

// Bitfield is a serializable bitmap for bittorrent
type Bitfield struct {
	// length in bits
	Length uint32 `bencode:"bits"`
	// bitfield data
	Data []byte `bencode:"bitfield"`
}

// NewBitfield creates new bitfield given number of bits and initial value
func NewBitfield(bits uint32, value []byte) *Bitfield {
	if value == nil {
		value = make([]byte, (bits/8)+1)
	}
	b := make([]byte, len(value))
	copy(b, value)
	return &Bitfield{
		Length: bits,
		Data:   b,
	}
}

// Inverted gets copy of current Bitfield with all bits inverted
func (bf *Bitfield) Inverted() (i *Bitfield) {
	i = NewBitfield(bf.Length, nil)
	bit := uint32(0)
	for bit < bf.Length {
		if !bf.Has(bit) {
			i.Set(bit)
		}
		bit++
	}
	return
}

// AND returns copy of Bitfield with bitwise AND applied from other Bitfield
func (bf *Bitfield) AND(other *Bitfield) *Bitfield {
	if bf.Length == other.Length {
		b := NewBitfield(bf.Length, bf.Data)
		for idx := range other.Data {
			b.Data[idx] &= other.Data[idx]
		}
		return b
	}
	return nil
}

// Equals checks if a Bitfield is equal to anoter
func (bf *Bitfield) Equals(other *Bitfield) bool {
	return bytes.Equal(bf.Data, other.Data)
}

// BEncode for fs storage
func (bf *Bitfield) BEncode(w io.Writer) (err error) {
	enc := bencode.NewEncoder(w)
	err = enc.Encode(bf)
	return
}

// BDecode for fs storage
func (bf *Bitfield) BDecode(r io.Reader) (err error) {
	dec := bencode.NewDecoder(r)
	err = dec.Decode(bf)
	return
}

// ToWireMessage serializes to bittorrent wire message
func (bf *Bitfield) ToWireMessage() *common.WireMessage {
	return common.NewWireMessage(common.BitField, bf.Data[:])
}

// Set sets a big at index
func (bf *Bitfield) Set(index uint32) {
	dl := uint32(len(bf.Data))
	if index < bf.Length {
		idx := index >> 3
		if idx < dl {
			bf.Data[idx] |= (1 << (7 - uint(index)&7))
		}
	}
}

// CountSet counts how many bits are set
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

// Has returns true if we have a bit at index set
func (bf *Bitfield) Has(index uint32) bool {
	dl := uint32(len(bf.Data))
	if index < bf.Length {
		idx := index >> 3
		if idx < dl {
			return bf.Data[idx]&(1<<(7-uint(index)&7)) != 0
		}
	}
	return false
}

// Completed returns true if this Bitfield is 100% set
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
