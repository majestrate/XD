package bittorrent

import (
	"io"
	"xd/lib/common"
	"xd/lib/util"
)

const _handshake_v1 = "BitTorrent protocol"



// bittorrent protocol handshake
type Handshake struct {
	Infohash common.Infohash
	PeerID common.PeerID
}

// recv handshake via reader
func (h *Handshake) Recv(r io.Reader) (err error) {
	var buff [68]byte
	_, err = io.ReadFull(r, buff[:])
	if err == nil {
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
	copy(buff[28:48], h.Infohash[:])
	copy(buff[48:68], h.PeerID[:])
	err = util.WriteFull(w, buff[:])
	return
}
