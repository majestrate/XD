package common

import (
	"encoding/binary"
	"io"
	"xd/lib/util"
)

// type for wire message id
type WireMessageType byte

// choke message
const Choke = WireMessageType(0)

// unchoke message
const UnChoke = WireMessageType(1)

// peer is interested message
const Interested = WireMessageType(2)

// peer is not interested message
const NotInterested = WireMessageType(3)

// have message
const Have = WireMessageType(4)

// bitfield message
const BitField = WireMessageType(5)

// request piece message
const Request = WireMessageType(6)

// response to REQUEST message
const Piece = WireMessageType(7)

// cancel a REQUEST message
const Cancel = WireMessageType(8)

// extended options message
const Extended = WireMessageType(20)

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
	case Extended:
		return "Extended"
	default:
		return "???"
	}
}

// bittorrent wire message
type WireMessage struct {
	data []byte
}

func KeepAlive() *WireMessage {
	return &WireMessage{
		data: []byte{0, 0, 0, 0},
	}
}

// create new wire message
func NewWireMessage(id WireMessageType, body []byte) *WireMessage {
	if body == nil {
		body = []byte{}
	}
	l := uint32(len(body)) + 5
	msg := &WireMessage{
		data: make([]byte, l),
	}
	binary.BigEndian.PutUint32(msg.data, l-4)
	msg.data[4] = byte(id)
	if len(body) > 0 {
		copy(msg.data[5:], body)
	}
	return msg
}

// return true if this message is a keepalive message
func (msg *WireMessage) KeepAlive() bool {
	return len(msg.data) == 4
}

// return the length of the body of this message
func (msg *WireMessage) Len() uint32 {
	return binary.BigEndian.Uint32(msg.data)
}

// return the body of this message
func (msg *WireMessage) Payload() []byte {
	if msg.Len() > 0 {
		return msg.data[5:]
	} else {
		return nil
	}
}

// return the id of this message (aka the type of message this is)
func (msg *WireMessage) MessageID() WireMessageType {
	return WireMessageType(msg.data[4])
}

// read message from reader
func (msg *WireMessage) Recv(r io.Reader) (err error) {
	// read header
	_, err = io.ReadFull(r, msg.data[:4])
	if err == nil {
		l := binary.BigEndian.Uint32(msg.data[:])
		if l > 0 {
			data := make([]byte, 4+l)
			binary.BigEndian.PutUint32(data[:], l)
			// read body
			_, err = io.ReadFull(r, data[4:])
			msg.data = data
		}
	}
	return
}

// send via writer
func (msg *WireMessage) Send(w io.Writer) (err error) {
	err = util.WriteFull(w, msg.data[:])
	return
}

func (p *PieceData) ToWireMessage() *WireMessage {
	body := make([]byte, len(p.Data)+8)
	copy(body[8:], p.Data)
	binary.BigEndian.PutUint32(body[:], p.Index)
	binary.BigEndian.PutUint32(body[4:], p.Begin)
	return NewWireMessage(Piece, body)
}

// convert piece request to wire message
func (req *PieceRequest) ToWireMessage() *WireMessage {
	var body [12]byte
	binary.BigEndian.PutUint32(body[:], req.Index)
	binary.BigEndian.PutUint32(body[4:], req.Begin)
	binary.BigEndian.PutUint32(body[8:], req.Length)
	return NewWireMessage(Request, body[:])
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
		p.Data = data[8:]
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

// get as have message
func (msg *WireMessage) GetHave() (h uint32) {
	if msg.MessageID() == Have {
		data := msg.Payload()
		if len(data) == 4 {
			h = binary.BigEndian.Uint32(data[:])
		}
	}
	return
}

// create new have message
func NewHave(idx uint32) *WireMessage {
	var body [4]byte
	binary.BigEndian.PutUint32(body[:], idx)
	return NewWireMessage(Have, body[:])
}

func NewNotInterested() *WireMessage {
	return NewWireMessage(NotInterested, nil)
}

func NewInterested() *WireMessage {
	return NewWireMessage(Interested, nil)
}
