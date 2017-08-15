package extensions

import (
	"bytes"
	"xd/lib/common"
	"xd/lib/version"

	"github.com/zeebo/bencode"
)

// Extension is a bittorrent extenension string
type Extension string

var extensionDefaults = map[Extension]uint8{
	I2PDHT:       1,
	PeerExchange: 2,
}

func (ex Extension) String() string {
	return string(ex)
}

// ExtendedOptions is a serializable BitTorrent extended options message
type Message struct {
	ID         uint8            `bencode:"-"`
	Version    string           `bencode:"v"` // handshake data
	Extensions map[string]uint8 `bencode:"m"` // handshake data
	Payload    interface{}
}

// supports PEX?
func (opts *Message) PEX() bool {
	return opts.IsSupported(PeerExchange.String())
}

func (opts *Message) SetSupported(ext Extension) {
	// TODO: this will error if we do not support this extension
	opts.Extensions[ext.String()] = extensionDefaults[ext]
}

func (opts *Message) IsSupported(ext string) (has bool) {
	_, has = opts.Extensions[ext]
	return
}

func (opts *Message) Lookup(id uint8) (string, bool) {
	for k, v := range opts.Extensions {
		if v == id {
			return k, true
		}
	}
	return "", false
}

// Copy makes a copy of this ExtendedOptions
func (opts *Message) Copy() *Message {
	ext := make(map[string]uint8)
	for k, v := range opts.Extensions {
		ext[k] = v
	}
	return &Message{
		ID:         opts.ID,
		Version:    opts.Version,
		Extensions: ext,
		Payload:    opts.Payload,
	}
}

// ToWireMessage serializes this ExtendedOptions to a BitTorrent wire message
func (opts *Message) ToWireMessage() *common.WireMessage {
	b := new(bytes.Buffer)
	b.Write([]byte{opts.ID})
	if opts.ID == 0 {
		bencode.NewEncoder(b).Encode(opts)
	} else if opts.Payload != nil {
		bencode.NewEncoder(b).Encode(opts.Payload)
	}
	return common.NewWireMessage(common.Extended, b.Bytes())
}

// New creates new valid ExtendedOptions instance
func New() *Message {
	return &Message{
		Version:    version.Version(),
		Extensions: make(map[string]uint8),
	}
}

func NewPEX(id uint8, connected, disconnected []byte) *Message {
	payload := map[string]interface{}{
		"added":   connected,
		"dropped": disconnected,
	}
	msg := New()
	msg.ID = id
	msg.Payload = payload
	return msg
}

// FromWireMessage loads an ExtendedOptions messgae from a BitTorrent wire message
func FromWireMessage(msg *common.WireMessage) (opts *Message) {
	if msg.MessageID() == common.Extended {
		payload := msg.Payload()
		if len(payload) > 0 {
			opts = &Message{
				ID: payload[0],
			}
			if opts.ID == 0 {
				// handshake
				bencode.DecodeBytes(payload[1:], opts)
			} else {
				// extension data
				bencode.DecodeBytes(payload[1:], &opts.Payload)
			}
		}
	}
	return
}
