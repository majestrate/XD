package extensions

import (
	"errors"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/util"
	"github.com/majestrate/XD/lib/version"
	"github.com/zeebo/bencode"
)

// Extension is a bittorrent extenension string
type Extension string

// String gets extension as string
func (ex Extension) String() string {
	return string(ex)
}

// Message is a serializable BitTorrent extended options message
type Message struct {
	ID           uint8             `bencode:"-"`
	Version      string            `bencode:"v"` // handshake data
	Extensions   map[string]uint32 `bencode:"m"` // handshake data
	Payload      interface{}       `bencode:"-"`
	PayloadRaw   []byte            `bencode:"-"`
	MetainfoSize *uint32           `bencode:"metadata_size,omitempty"`
}

// I2PPEX returns true if i2p PEX is supported
func (opts Message) I2PPEX() bool {
	return opts.IsSupported(I2PPeerExchange.String())
}

// LNPEX returns true if we support lokinet pex
func (opts Message) LNPEX() bool {
	return opts.IsSupported(LokinetPeerExchange.String())
}

// XDHT returns true if XHDT is supported
func (opts Message) XDHT() bool {
	return opts.IsSupported(XDHT.String())
}

// MetaData returns true if ut_metadata is supported
func (opts Message) MetaData() bool {
	return opts.IsSupported(UTMetaData.String())
}

// SetSupported sets a bittorrent extension as supported
func (opts *Message) SetSupported(ext Extension) {
	// get next id
	nextId := uint32(1)
	for k, v := range opts.Extensions {
		if v >= nextId {
			nextId = v + 1
		}
		// already supported
		if k == ext.String() {
			return
		}
	}
	// set supported
	opts.Extensions[ext.String()] = nextId
}

// IsSupported returns true if an extension by its name is supported
func (opts Message) IsSupported(ext string) (has bool) {
	if opts.Extensions != nil {
		_, has = opts.Extensions[ext]
	}
	return
}

// Lookup finds the extension name of the extension by id
func (opts Message) Lookup(id uint8) (string, bool) {
	for k, v := range opts.Extensions {
		if v == uint32(id) {
			return k, true
		}
	}
	return "", false
}

// Copy makes a copy of this Message
func (opts Message) Copy() Message {
	ext := make(map[string]uint32)
	for k, v := range opts.Extensions {
		ext[k] = v
	}
	m := Message{
		ID:           opts.ID,
		Version:      opts.Version,
		Extensions:   ext,
		Payload:      opts.Payload,
		MetainfoSize: opts.MetainfoSize,
	}
	if opts.PayloadRaw != nil {
		m.PayloadRaw = make([]byte, len(opts.PayloadRaw))
		copy(m.PayloadRaw, opts.PayloadRaw)
	}
	return m
}

// ToWireMessage serializes this ExtendedOptions to a BitTorrent wire message
func (opts Message) ToWireMessage() common.WireMessage {
	var body []byte
	if opts.ID == 0 {
		var b util.Buffer
		bencode.NewEncoder(&b).Encode(opts)
		body = b.Bytes()
	} else if opts.Payload != nil {
		var b util.Buffer
		bencode.NewEncoder(&b).Encode(opts.Payload)
		body = b.Bytes()
	} else if opts.PayloadRaw != nil {
		body = opts.PayloadRaw
	} else {
		// wtf? invalid message
		return nil
	}
	return common.NewWireMessage(common.Extended, []byte{opts.ID}, body)
}

// New creates new valid Message instance
func New() Message {
	return Message{
		Version:    version.Version(),
		Extensions: make(map[string]uint32),
	}
}

// NewOur creates a new Message instance with metadata size set
func NewOur(sz uint32) Message {
	m := Message{
		Version:    version.Version(),
		Extensions: make(map[string]uint32),
	}
	if sz > 0 {
		m.MetainfoSize = &sz
	}
	return m
}

// NewI2PPEX creates a new PEX message for i2p peers
func NewI2PPEX(id uint8, connected, disconnected []byte) Message {
	payload := map[string]interface{}{
		"added":   connected,
		"dropped": disconnected,
	}
	msg := New()
	msg.ID = id
	msg.Payload = payload
	return msg
}

// NewLNPex creates a new PEX message for lokinet peers
func NewLNPEX(id uint8, connected, disconnected []common.Peer) Message {
	payload := map[string]interface{}{
		"added":   connected,
		"dropped": disconnected,
	}
	msg := New()
	msg.ID = id
	msg.Payload = payload
	return msg
}

var ErrInvalidSize = errors.New("invalid message size")
var ErrInvalidMessageID = errors.New("invalid message id")

// FromWireMessage loads an ExtendedOptions messgae from a BitTorrent wire message
func FromWireMessage(msg common.WireMessage) (opts Message, err error) {
	if msg.MessageID() == common.Extended {
		payload := msg.Payload()
		if len(payload) > 1 {
			opts = Message{
				ID:         payload[0],
				PayloadRaw: payload[1:],
			}
			if opts.ID == 0 {
				// handshake
				bencode.DecodeBytes(opts.PayloadRaw, &opts)
				// clear out raw payload because handshake
				opts.PayloadRaw = nil
			} else {
				bencode.DecodeBytes(opts.PayloadRaw, &opts.Payload)
			}
		} else {
			err = ErrInvalidSize
		}
	} else {
		err = ErrInvalidMessageID
	}
	return
}
