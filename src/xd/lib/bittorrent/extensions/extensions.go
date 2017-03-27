package extensions

import (
	"bytes"
	"xd/lib/common"
	"xd/lib/version"

	"github.com/zeebo/bencode"
)

// Extension is a bittorrent extenension string
type Extension string

// PeerExchange is a BitTorrent Extension indicating we support PEX
const PeerExchange = Extension("ut_pex")

func (ex Extension) String() string {
	return string(ex)
}

// ExtendedOptions is a serializable BitTorrent extended options message
type ExtendedOptions struct {
	ID         uint8               `bencode:"-"`
	Version    string              `bencode:"v"`
	Extensions map[Extension]uint8 `bencode:"m"`
}

// Copy makes a copy of this ExtendedOptions
func (opts *ExtendedOptions) Copy() *ExtendedOptions {
	ext := make(map[Extension]uint8)
	for k, v := range opts.Extensions {
		ext[k] = v
	}
	return &ExtendedOptions{
		ID:         opts.ID,
		Version:    opts.Version,
		Extensions: ext,
	}
}

// ToWireMessage serializes this ExtendedOptions to a BitTorrent wire message
func (opts *ExtendedOptions) ToWireMessage() *common.WireMessage {
	b := new(bytes.Buffer)
	b.Write([]byte{opts.ID})
	if opts.ID == 0 {
		bencode.NewEncoder(b).Encode(opts)
	}
	return common.NewWireMessage(common.Extended, b.Bytes())
}

// New creates new valid ExtendedOptions instance
func New() *ExtendedOptions {
	return &ExtendedOptions{
		Version:    version.Version,
		Extensions: make(map[Extension]uint8),
	}
}

// FromWireMessage loads an ExtendedOptions messgae from a BitTorrent wire message
func FromWireMessage(msg *common.WireMessage) (opts *ExtendedOptions) {
	if msg.MessageID() == common.Extended {
		payload := msg.Payload()
		if len(payload) > 0 {
			opts = &ExtendedOptions{
				ID: payload[0],
			}
			bencode.DecodeBytes(payload[1:], opts)
		}
	}
	return
}
