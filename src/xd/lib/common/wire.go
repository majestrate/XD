package common

import (
	"encoding/binary"
	"io"
	"xd/lib/util"
)

// WireMessageType is type for wire message id
type WireMessageType byte

// Chock is message id for choke message
const Choke = WireMessageType(0)

// UnChoke is message id for unchoke message
const UnChoke = WireMessageType(1)

// Interested is messageid for interested message
const Interested = WireMessageType(2)

// NotInterested is messageid for not-interested message
const NotInterested = WireMessageType(3)

// Have is messageid for have message
const Have = WireMessageType(4)

// BitField is messageid for bitfield message
const BitField = WireMessageType(5)

// Request is messageid for piece request message
const Request = WireMessageType(6)

// Piece is messageid for response to Request message
const Piece = WireMessageType(7)

// Cancel is messageid for a Cancel message, used to cancel a pending request
const Cancel = WireMessageType(8)

// Extended is messageid for ExtendedOptions message
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

// WireMessage is a serializable bittorrent wire message
type WireMessage struct {
	data []byte
}

// KeepAlive makes a WireMessage of size 0
func KeepAlive() *WireMessage {
	return &WireMessage{
		data: []byte{0, 0, 0, 0},
	}
}

// NewWireMessage creates new wire message with id and body
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

// KeepAlive returns true if this message is a keepalive message
func (msg *WireMessage) KeepAlive() bool {
	return msg.Len() == 0
}

// Len returns the length of the body of this message
func (msg *WireMessage) Len() uint32 {
	return binary.BigEndian.Uint32(msg.data)
}

// Payload returns a byteslice for the body of this message
func (msg *WireMessage) Payload() []byte {
	if msg.Len() > 0 {
		return msg.data[5:]
	} else {
		return nil
	}
}

// MessageID returns the id of this message
func (msg *WireMessage) MessageID() WireMessageType {
	return WireMessageType(msg.data[4])
}

// Recv reads message from reader
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

// Send writes WireMessage via writer
func (msg *WireMessage) Send(w io.Writer) (err error) {
	err = util.WriteFull(w, msg.data[:])
	return
}

// ToWireMessage serialize to BitTorrent wire message
func (p *PieceData) ToWireMessage() *WireMessage {
	body := make([]byte, len(p.Data)+8)
	copy(body[8:], p.Data[:])
	binary.BigEndian.PutUint32(body[:], p.Index)
	binary.BigEndian.PutUint32(body[4:], p.Begin)
	return NewWireMessage(Piece, body)
}

// ToWireMessage serialize to BitTorrent wire message
func (req *PieceRequest) ToWireMessage() *WireMessage {
	var body [12]byte
	binary.BigEndian.PutUint32(body[:], req.Index)
	binary.BigEndian.PutUint32(body[4:], req.Begin)
	binary.BigEndian.PutUint32(body[8:], req.Length)
	return NewWireMessage(Request, body[:])
}

// GetPieceData gets this wire message as a PieceData if applicable
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

// GetPieceRequest gets piece request from wire message or nil if malformed or not a piece request
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

// GetHave gets the piece index of a have message
func (msg *WireMessage) GetHave() (h uint32) {
	if msg.MessageID() == Have {
		data := msg.Payload()
		if len(data) == 4 {
			h = binary.BigEndian.Uint32(data[:])
		}
	}
	return
}

// NewHave creates a new have message
func NewHave(idx uint32) *WireMessage {
	var body [4]byte
	binary.BigEndian.PutUint32(body[:], idx)
	return NewWireMessage(Have, body[:])
}

// NewNotInterested creates a new NotInterested message
func NewNotInterested() *WireMessage {
	return NewWireMessage(NotInterested, nil)
}

// NewInterested creates a new Interested message
func NewInterested() *WireMessage {
	return NewWireMessage(Interested, nil)
}
