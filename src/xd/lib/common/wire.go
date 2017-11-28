package common

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"xd/lib/log"
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

// HashReqeuest (v2)
const HashRequest = WireMessageType(21)

// Hashes (v2)
const Hashes = WireMessageType(22)

// HashReject (v2)
const HashReject = WireMessageType(23)

// special for invalid
const Invalid = WireMessageType(255)

// String returns a string name of this wire message id
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
	case HashRequest:
		return "HashRequest"
	case Hashes:
		return "Hashes"
	case HashReject:
		return "HashReject"
	case Invalid:
		return "INVALID"
	default:
		return fmt.Sprintf("??? (%d)", uint8(t))
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
func NewWireMessage(id WireMessageType, body []byte) (msg *WireMessage) {
	if body == nil {
		body = []byte{}
	}
	var hdr [4]byte
	l := uint32(len(body))
	msg = new(WireMessage)
	binary.BigEndian.PutUint32(hdr[:], l+1)
	msg.data = append(hdr[:], byte(id))
	msg.data = append(msg.data, body...)
	return msg
}

// read wire messages from reader and call a function on each it gets
// reads until reader is done
func ReadWireMessages(r io.Reader, f func(*WireMessage) error) (err error) {
	for err == nil {
		var msg WireMessage
		msg.data = []byte{0, 0, 0, 0}
		_, err = io.ReadFull(r, msg.data)
		l := binary.BigEndian.Uint32(msg.data[:])
		if l > 0 {
			var body []byte
			lr := &io.LimitedReader{R: r, N: int64(l)}
			log.Debugf("read message of size %d bytes", l)
			body, err = ioutil.ReadAll(lr)
			if lr.N == 0 {
				msg.data = append(msg.data, body...)
				err = f(&msg)
			}
		}
	}
	return
}

// KeepAlive returns true if this message is a keepalive message
func (msg *WireMessage) KeepAlive() bool {
	return msg.Len() == 0
}

// Len returns the length of the body of this message
func (msg *WireMessage) Len() uint32 {
	if msg.data == nil {
		return 0
	}
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
	if len(msg.data) > 4 {
		return WireMessageType(msg.data[4])
	}
	return Invalid
}

var ErrToBig = errors.New("message too big")

// Send writes WireMessage via writer
func (msg *WireMessage) Send(w io.Writer) (err error) {
	err = util.WriteFull(w, msg.data)
	return
}

// ToWireMessage serialize to BitTorrent wire message
func (p *PieceData) ToWireMessage() *WireMessage {
	var hdr [8]byte
	var body []byte
	binary.BigEndian.PutUint32(hdr[:], p.Index)
	binary.BigEndian.PutUint32(hdr[4:], p.Begin)
	body = append(hdr[:], p.Data...)
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
func (msg *WireMessage) GetPieceData() (p *PieceData) {

	if msg.MessageID() == Piece {
		data := msg.Payload()
		if len(data) > 8 {
			p = new(PieceData)
			p.Index = binary.BigEndian.Uint32(data)
			p.Begin = binary.BigEndian.Uint32(data[4:])
			p.Data = data[8:]
		}
	}
	return
}

// GetPieceRequest gets piece request from wire message
func (msg WireMessage) GetPieceRequest() (req *PieceRequest) {
	if msg.MessageID() == Request {
		data := msg.Payload()
		if len(data) == 12 {
			req = new(PieceRequest)
			req.Index = binary.BigEndian.Uint32(data[:])
			req.Begin = binary.BigEndian.Uint32(data[4:])
			req.Length = binary.BigEndian.Uint32(data[8:])
		}
	}
	return
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
