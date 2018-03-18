package util

import (
	"encoding/binary"
	"golang.org/x/crypto/blake2b"
	"io"
)

type bloomFilterGeneration struct {
	data []byte
}

type bloomFilterMark struct {
	idx  uint64
	mask byte
}

func (m bloomFilterMark) hit(data []byte) bool {
	return data[m.idx%uint64(len(data))]&m.mask == m.mask
}

const bloomFilterMarkSize = 9

func (gen *bloomFilterGeneration) getMark(data []byte) bloomFilterMark {
	var buff [bloomFilterMarkSize]byte
	x, _ := blake2b.NewXOF(bloomFilterMarkSize, nil)
	WriteFull(x, data)
	io.ReadFull(x, buff[:])
	return bloomFilterMark{
		idx:  binary.BigEndian.Uint64(buff[1:]),
		mask: 1 << (buff[0] % 8),
	}
}

func (gen *bloomFilterGeneration) putMark(mark bloomFilterMark) {
	gen.data[mark.idx%uint64(len(gen.data))] |= mark.mask
}

func (gen *bloomFilterGeneration) hasHit(data []byte) (mark bloomFilterMark, has bool) {

	mark = gen.getMark(data)
	has = mark.hit(gen.data)
	return
}

// BloomFilter is a decaying bloom filter
type BloomFilter struct {
	gen  uint
	gens []bloomFilterGeneration
}

// NewBloomFilter creates a new decaying bloom filter with gen Generations each genSize bytes large
func NewBloomFilter(gens, genSize int) *BloomFilter {
	f := &BloomFilter{
		gens: make([]bloomFilterGeneration, gens),
	}
	for idx := range f.gens {
		f.gens[idx].data = make([]byte, genSize)
	}
	return f
}

// Check checks for a bloom filter hit
// returns true on hit
// returns false and adds an entry for data on miss
func (f *BloomFilter) Check(data []byte) (hit bool) {
	var mark bloomFilterMark
	mark, hit = f.currentGen().hasHit(data)
	if hit {
		return
	}
	// check for hit in previous generation
	for idx := range f.gens {
		_, hit = f.gens[idx].hasHit(data)
		if hit {
			return
		}
	}
	f.currentGen().putMark(mark)
	return false
}

// Decay decays bloom filter a generation
func (f *BloomFilter) Decay() {
	f.gen++
}

func (f *BloomFilter) currentGen() *bloomFilterGeneration {
	return &f.gens[f.gen%uint(len(f.gens))]
}
