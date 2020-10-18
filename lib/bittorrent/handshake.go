package bittorrent

import (
	"bytes"
	"errors"
	"io"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/util"
)

const handshakeV1 = "BitTorrent protocol"

// Reserved is reserved data in handshake
type Reserved struct {
	data [8]uint8
}

// ReservedBit is a bit set in reserved data
type ReservedBit uint8

func (b ReservedBit) mask() uint8 {
	return 1 << (7 - (uint8(b-1) % 8))
}

func (b ReservedBit) index() uint8 {
	return uint8(b-1) / 8
}

// Has returns true if reserved bit is set
func (r Reserved) Has(bit ReservedBit) bool {
	return r.data[bit.index()]&bit.mask() == bit.mask()
}

// Set sets a reserved bit
func (r *Reserved) Set(bit ReservedBit) {
	r.data[bit.index()] |= bit.mask()
}

// Extension is ReservedBit for bittorrent extensions
const Extension = ReservedBit(44)

// DHT is ReservedBit for BT DHT
const DHT = ReservedBit(64)

// ErrInvalidHandshake is returned when a handshake contained invalid format
var ErrInvalidHandshake = errors.New("invalid bittorrent handshake")

// Handshake is a bittorrent protocol handshake info
type Handshake struct {
	Reserved Reserved
	Infohash common.Infohash
	PeerID   common.PeerID
}

// FromBytes parses bittorrent handshake from byteslice
func (h *Handshake) FromBytes(data []byte) (err error) {
	if len(data) < 68 {
		err = ErrInvalidHandshake
	} else {
		buff := data[:68]
		if buff[0] == 19 && bytes.Equal(buff[1:20], []byte(handshakeV1)) {
			copy(h.Reserved.data[:], buff[20:28])
			copy(h.Infohash[:], buff[28:48])
			copy(h.PeerID[:], buff[48:68])
		} else {
			err = ErrInvalidHandshake
		}
	}
	return
}

// Recv reads handshake via reader
func (h *Handshake) Recv(r io.Reader) (err error) {
	var buff [68]byte
	_, err = io.ReadFull(r, buff[:])
	if err == nil {
		err = h.FromBytes(buff[:])
	}
	return
}

// Send sends handshake via writer
func (h *Handshake) Send(w io.Writer) (err error) {
	var buff [68]byte
	buff[0] = 19
	copy(buff[1:], []byte(handshakeV1))
	copy(buff[20:28], h.Reserved.data[:])
	copy(buff[28:48], h.Infohash[:])
	copy(buff[48:68], h.PeerID[:])
	err = util.WriteFull(w, buff[:])
	return
}
