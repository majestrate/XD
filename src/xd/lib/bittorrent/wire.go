package bittorrent

import (
	"encoding/binary"
	"io"
	"xd/lib/util"
)

// type for wire message id
type WireMessageType byte

// keep alive message
const KeepAlive = WireMessageType(0)

// choke message
const Choke = WireMessageType(1)

// unchoke message
const UnChoke = WireMessageType(2)

// peer is interested message
const Interested = WireMessageType(3)

// peer is not interested message
const NotInterested = WireMessageType(4)

// have message
const Have = WireMessageType(5)

// bitfield message
const BitField = WireMessageType(6)

// request piece message
const Request = WireMessageType(7)

// response to REQUEST message
const Piece = WireMessageType(8)

// cancel a REQUEST message
const Cancel = WireMessageType(9)

// extention
const Port = WireMessageType(10)

func (t WireMessageType) String() string {
	switch t {
	case Choke:
		return "Choke"
	case UnChoke:
		return "UnChoke"
	case Interested:
		return "Interested"
	case NotInterested:
		return "NotInterested"
	case Have:
		return "Have"
	case BitField:
		return "BitField"
	case Request:
		return "Request"
	case Piece:
		return "Piece"
	case Cancel:
		return "Cancel"
	case Port:
		return "Port"
	default:
		return "???"
	}
}

// bittorrent wire message
type WireMessage struct {
	length uint32
	data   []byte
}

// create new wire message
func NewWireMessage(id WireMessageType, body []byte) *WireMessage {
	if body == nil {
		body = []byte{}
	}
	msg := &WireMessage{
		length: uint32(1 + len(body)),
	}
	msg.data = make([]byte, msg.length)
	msg.data[0] = byte(id)
	copy(msg.data[1:], body)
	return msg
}

// return true if this message is a keepalive message
func (msg *WireMessage) KeepAlive() bool {
	return msg.length == 0
}

// return the length of the body of this message
func (msg *WireMessage) Len() uint32 {
	return msg.length
}

// return the body of this message
func (msg *WireMessage) Payload() []byte {
	if msg.length > 0 {
		return msg.data[1:]
	} else {
		return nil
	}
}

// return the id of this message (aka the type of message this is)
func (msg *WireMessage) MessageID() WireMessageType {
	return WireMessageType(msg.data[0])
}

// read message from reader
func (msg *WireMessage) Recv(r io.Reader) (err error) {
	// read header
	var buff [4]byte
	_, err = io.ReadFull(r, buff[:])
	if err == nil {
		msg.length = binary.BigEndian.Uint32(buff[:])
		if msg.length > 0 {
			msg.data = make([]byte, int(msg.length))
			// read body
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

// a piece request
type PieceRequest struct {
	Index  uint32
	Begin  uint32
	Length uint32
}

// convert piece request to wire message
func (req *PieceRequest) ToWireMessage() *WireMessage {
	var body [12]byte
	binary.BigEndian.PutUint32(body[:], req.Index)
	binary.BigEndian.PutUint32(body[4:], req.Begin)
	binary.BigEndian.PutUint32(body[8:], req.Length)
	return NewWireMessage(Request, body[:])
}

type PieceData struct {
	Index uint32
	Begin uint32
	Data  []byte
}

func (msg *WireMessage) GetPieceData() *PieceData {

	if msg.MessageID() != Piece {
		return nil
	}
	data := msg.Payload()
	if len(data) > 8 {
		p := new(PieceData)
		p.Index = binary.BigEndian.Uint32(data)
		p.Begin = binary.BigEndian.Uint32(data[4:])
		p.Data = make([]byte, len(data)-8)
		copy(p.Data, data[8:])
		return p
	}
	return nil
}

// get piece request from wire message or nil if malformed or not a piece request
func (msg *WireMessage) GetPieceRequest() *PieceRequest {
	if msg.MessageID() != Request {
		return nil
	}
	req := new(PieceRequest)
	data := msg.Payload()
	if len(data) != 12 {
		return nil
	}
	req.Index = binary.BigEndian.Uint32(data[:])
	req.Begin = binary.BigEndian.Uint32(data[4:])
	req.Length = binary.BigEndian.Uint32(data[8:])
	return req
}
