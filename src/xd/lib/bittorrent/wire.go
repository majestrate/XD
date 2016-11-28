package bittorrent

import (
	"encoding/binary"
	"io"
	"xd/lib/util"
)

const CHOKE = 0
const UNCHOKE = 1
const INTERESTED = 2
const NOT_INTERESTED = 3
const HAVE = 4
const BITFIELD = 5
const REQUEST = 6
const PIECE = 7
const CANCEL = 8
const PORT = 9


// bittorrent wire message
type WireMessage struct {
	length uint32 
	data []byte
}

// create new wire message
func NewWireMessage(id byte, body []byte) *WireMessage {
	msg := &WireMessage{
		length: uint32(1 + len(body)),
	}
	msg.data = make([]byte, msg.length)
	msg.data[0] = id
	copy(msg.data[1:], body)
	return msg
}

func (msg *WireMessage) KeepAlive() bool {
	return msg.length == 0
}

func (msg *WireMessage) Len() uint32 {
	return msg.length
}

func (msg *WireMessage) Payload() []byte {
	return msg.data[1:]
}

func (msg *WireMessage) MessageID() byte {
	return msg.data[0]
}

// recv from reader
func (msg *WireMessage) Recv(r io.Reader) (err error) {
	var buff [4]byte
	_, err = io.ReadFull(r, buff[:])
	if err == nil {
		msg.length = binary.BigEndian.Uint32(buff[:])
		if msg.length > 0 {
			msg.data = make([]byte, int(msg.length))
			_, err = io.ReadFull(r, msg.data)
		}
	}
	return
}

// send via writer
func (msg *WireMessage) Send(w io.Writer) (err error) {
	var buff [4]byte
	binary.BigEndian.PutUint32(buff[:], msg.length)
	err = util.WriteFull(w, buff[:])
	if err == nil && msg.length > 0 {
		err = util.WriteFull(w, msg.data)
	}
	return
}
