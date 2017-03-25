package bittorrent

import (
	"io"
	"xd/lib/common"
	"xd/lib/util"
)

const _handshake_v1 = "BitTorrent protocol"

// reserved data
type Reserved struct {
	data [8]uint8
}

// bit set in reserved data
type ReservedBit uint8

func (b ReservedBit) mask() uint8 {
	return 1 << (7 - (uint8(b-1) % 8))
}

func (b ReservedBit) index() uint8 {
	return uint8(b-1) / 8
}

// return true if reserved bit is set
func (r Reserved) Has(bit ReservedBit) bool {
	return r.data[bit.index()]&bit.mask() == bit.mask()
}

// set a reserved bit
func (r *Reserved) Set(bit ReservedBit) {
	r.data[bit.index()] |= bit.mask()
}

const Extension = ReservedBit(44)
const DHT = ReservedBit(64)

// bittorrent protocol handshake info
type Handshake struct {
	Reserved Reserved
	Infohash common.Infohash
	PeerID   common.PeerID
}

// recv handshake via reader
func (h *Handshake) Recv(r io.Reader) (err error) {
	var buff [68]byte
	_, err = io.ReadFull(r, buff[:])
	if err == nil {
		copy(h.Reserved.data[:], buff[20:28])
		copy(h.Infohash[:], buff[28:48])
		copy(h.PeerID[:], buff[48:68])
	}
	return
}

// send handshake via writer
func (h *Handshake) Send(w io.Writer) (err error) {
	var buff [68]byte
	buff[0] = 19
	copy(buff[1:], []byte(_handshake_v1))
	copy(buff[20:28], h.Reserved.data[:])
	copy(buff[28:48], h.Infohash[:])
	copy(buff[48:68], h.PeerID[:])
	err = util.WriteFull(w, buff[:])
	return
}
