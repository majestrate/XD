package extensions

import (
	"bytes"
	"github.com/zeebo/bencode"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/version"
)

// Extension is a bittorrent extenension string
type Extension string

var extensionDefaults = map[Extension]uint32{
	//I2PDHT:       1,
	PeerExchange: 2,
	XDHT:         3,
	UTMetaData:   4,
}

func (ex Extension) String() string {
	return string(ex)
}

// ExtendedOptions is a serializable BitTorrent extended options message
type Message struct {
	ID           uint8             `bencode:"-"`
	Version      string            `bencode:"v"` // handshake data
	Extensions   map[string]uint32 `bencode:"m"` // handshake data
	Payload      interface{}       `bencode:"-"`
	PayloadRaw   []byte            `bencode:"-"`
	MetainfoSize *uint32           `bencode:"metadata_size",omitempty`
}

// supports PEX?
func (opts *Message) PEX() bool {
	return opts.IsSupported(PeerExchange.String())
}

// supports XDHT
func (opts *Message) XDHT() bool {
	return opts.IsSupported(XDHT.String())
}

// supports ut_metadata
func (opts *Message) MetaData() bool {
	return opts.IsSupported(UTMetaData.String())
}

// set a bittorrent extension as supported
func (opts *Message) SetSupported(ext Extension) {
	// TODO: this will error if we do not support this extension
	opts.Extensions[ext.String()] = extensionDefaults[ext]
}

// return true if an extension by its name is supported
func (opts *Message) IsSupported(ext string) (has bool) {
	_, has = opts.Extensions[ext]
	return
}

// Lookup finds the extension name of the extension by id
func (opts *Message) Lookup(id uint8) (string, bool) {
	for k, v := range opts.Extensions {
		if v == uint32(id) {
			return k, true
		}
	}
	return "", false
}

// Copy makes a copy of this ExtendedOptions
func (opts *Message) Copy() *Message {
	ext := make(map[string]uint32)
	for k, v := range opts.Extensions {
		ext[k] = v
	}
	return &Message{
		ID:           opts.ID,
		Version:      opts.Version,
		Extensions:   ext,
		Payload:      opts.Payload,
		MetainfoSize: opts.MetainfoSize,
	}
}

// ToWireMessage serializes this ExtendedOptions to a BitTorrent wire message
func (opts *Message) ToWireMessage() common.WireMessage {
	b := new(bytes.Buffer)
	b.Write([]byte{opts.ID})
	if opts.ID == 0 {
		bencode.NewEncoder(b).Encode(opts)
	} else if opts.Payload != nil {
		bencode.NewEncoder(b).Encode(opts.Payload)
	} else if opts.PayloadRaw != nil {
		b.Write(opts.PayloadRaw)
	}
	log.Debugf("extended bytes %q", b.Bytes())
	return common.NewWireMessage(common.Extended, b.Bytes())
}

// New creates new valid ExtendedOptions instance
func New() *Message {
	return &Message{
		Version:    version.Version(),
		Extensions: make(map[string]uint32),
	}
}

func NewOur(sz uint32) *Message {
	m := &Message{
		Version: version.Version(),
		Extensions: map[string]uint32{
			UTMetaData.String():   3,
			PeerExchange.String(): 4,
		},
	}
	if sz > 0 {
		m.MetainfoSize = &sz
	}
	return m
}

// NewPEX creates a new PEX message for i2p peers
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
func FromWireMessage(msg common.WireMessage) (opts *Message) {
	if msg.MessageID() == common.Extended {
		payload := msg.Payload()
		if len(payload) > 0 {
			body := make([]byte, len(payload))
			copy(body, payload)
			opts = &Message{
				ID:         body[0],
				PayloadRaw: body[1:],
			}
			if opts.ID == 0 {
				// handshake
				bencode.DecodeBytes(opts.PayloadRaw, opts)
			} else {
				bencode.DecodeBytes(opts.PayloadRaw, &opts.Payload)
			}
		}
	}
	return
}
