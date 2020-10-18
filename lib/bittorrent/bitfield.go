package bittorrent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"github.com/majestrate/XD/lib/common"

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

// Copy returns an immutable copy of this bitfield
func (bf *Bitfield) Copy() *Bitfield {
	return NewBitfield(bf.Length, bf.Data)
}

// CopyFrom copies state from other into itself
func (bf *Bitfield) CopyFrom(other *Bitfield) {
	bf.Length = other.Length
	bf.Data = make([]byte, len(other.Data))
	copy(bf.Data, other.Data)
}

// UnmarshalJSON implements json.Marhsaller
func (bf *Bitfield) UnmarshalJSON(data []byte) (err error) {
	var bl []int
	err = json.Unmarshal(data, &bl)
	if err == nil {
		bf.Length = uint32(len(bl))
		bf.Data = make([]byte, (bf.Length/8)+1)
		for idx, v := range bl {
			if v != 0 {
				bf.Set(uint32(idx))
			}
		}
	}
	return
}

// MarshalJSON implements json.Marshaller
func (bf Bitfield) MarshalJSON() (data []byte, err error) {
	var ls []int
	idx := uint32(0)
	for idx < bf.Length {
		if bf.Has(idx) {
			ls = append(ls, 1)
		} else {
			ls = append(ls, 0)
		}
		idx++
	}
	data, err = json.Marshal(ls)
	return
}

// Zero sets all bits to zero
func (bf *Bitfield) Zero() {
	bf.Data = make([]byte, (bf.Length/8)+1)
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

// SelfOR applies bitwise OR from other Bitfield to itself
func (bf *Bitfield) SelfOR(other *Bitfield) {
	if bf.Length == other.Length {
		for idx := range other.Data {
			if idx < len(bf.Data) {
				bf.Data[idx] |= other.Data[idx]
			}
		}
	}
}

// OR returns Bitfield with bitwise OR applied from other Bitfield
func (bf *Bitfield) OR(other *Bitfield) *Bitfield {
	if bf.Length == other.Length {
		b := NewBitfield(bf.Length, bf.Data)
		for idx := range other.Data {
			if idx < len(b.Data) {
				b.Data[idx] |= other.Data[idx]
			}
		}
		return b
	}
	return nil
}

// XOR returns Bitfield with bitwise XOR applied from other Bitfield
func (bf *Bitfield) XOR(other *Bitfield) *Bitfield {
	if bf.Length == other.Length {
		b := NewBitfield(bf.Length, bf.Data)
		for idx := range other.Data {
			if idx < len(b.Data) {
				b.Data[idx] ^= other.Data[idx]
			}
		}
		return b
	}
	return nil
}

// Progress returns precent done as a float between 0 and 1
func (bf *Bitfield) Progress() (fl float64) {
	if bf.Length > 0 {
		fl = float64(bf.CountSet())
		fl /= float64(bf.Length)
	}
	return fl
}

// Percent returns string represnetation of percent done
func (bf *Bitfield) Percent() string {
	fl := float64(bf.CountSet())
	fl /= float64(bf.Length)
	return fmt.Sprintf("%.2f%%", fl*100)
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
func (bf *Bitfield) ToWireMessage() common.WireMessage {
	return common.NewWireMessage(common.BitField, bf.Data[:])
}

// Unset unsets a big at index
func (bf *Bitfield) Unset(index uint32) {
	dl := uint32(len(bf.Data))
	if index < bf.Length {
		idx := index >> 3
		if idx < dl {
			bf.Data[idx] &= ^(1 << (7 - uint(index)&7))
		}
	}
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

type rareSet map[uint32]uint32

// FindRarest finds the set bit we have that is rarest in others
func (bf *Bitfield) FindRarest(others []*Bitfield, exclude func(uint32) bool) (idx uint32, has bool) {
	bits := make(rareSet)
	i := bf.Length

	for i > 0 {
		i--
		bits[i] = 0
		for _, other := range others {
			if other.Has(i) {
				bits[i]++
			}
		}
	}

	min := ^uint32(0)
	for index, count := range bits {
		if exclude(index) {
			continue
		}
		if count < min && bf.Has(index) {
			min = count
			idx = index
			has = true
		}
	}
	return

}
