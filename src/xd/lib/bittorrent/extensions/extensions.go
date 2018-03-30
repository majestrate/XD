package extensions

import (
	"errors"
	"github.com/zeebo/bencode"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/util"
	"xd/lib/version"
)

// Extension is a bittorrent extenension string
type Extension string

var extensionDefaults = map[string]uint32{
	//I2PDHT:       1,
	PeerExchange.String(): 2,
	XDHT.String():         3,
	UTMetaData.String():   4,
}

// String gets extension as string
func (ex Extension) String() string {
	return string(ex)
}

// Message is a serializable BitTorrent extended options message
type Message struct {
	ID           uint8             `bencode:"-"`
	Version      string            `bencode:"v"` // handshake data
	Extensions   map[string]uint32 `bencode:"m"` // handshake data
	payload      interface{}       `bencode:"-"`
	Raw          []byte            `bencode:"-"`
	MetainfoSize *uint32           `bencode:"metadata_size,omitempty"`
}

// DecodePayload decodes the extended message's bytes into i
func (opts Message) DecodePayload(i interface{}) (err error) {
	err = bencode.DecodeBytes(opts.Raw, i)
	return
}

// PEX returns true if PEX is supported
func (opts Message) PEX() bool {
	return opts.IsSupported(PeerExchange.String())
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
	// TODO: this will error if we do not support this extension
	opts.Extensions[ext.String()] = extensionDefaults[ext.String()]
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
		payload:      opts.payload,
		MetainfoSize: opts.MetainfoSize,
	}
	if opts.Raw != nil {
		m.Raw = make([]byte, len(opts.Raw))
		copy(m.Raw, opts.Raw)
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
	} else if opts.payload != nil {
		var b util.Buffer
		bencode.NewEncoder(&b).Encode(opts.payload)
		body = b.Bytes()
	} else if opts.Raw != nil {
		body = opts.Raw
	} else {
		// wtf? invalid message
		log.Errorf("cannot create invalid extended message: %q", opts)
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
		Extensions: extensionDefaults,
	}
	if sz > 0 {
		m.MetainfoSize = &sz
	}
	return m
}

// NewPEX creates a new PEX message for i2p peers
func NewPEX(id uint8, connected, disconnected []byte) Message {
	payload := map[string]interface{}{
		"added":   connected,
		"dropped": disconnected,
	}
	msg := New()
	msg.ID = id
	msg.payload = payload
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
				ID:  payload[0],
				Raw: payload[1:],
			}
			if opts.ID == 0 {
				// handshake
				bencode.DecodeBytes(opts.Raw, &opts)
			}
		} else {
			err = ErrInvalidSize
		}
	} else {
		err = ErrInvalidMessageID
	}
	return
}
