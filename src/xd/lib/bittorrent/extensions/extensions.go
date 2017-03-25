package extensions

import (
	"bytes"
	"github.com/zeebo/bencode"
	"xd/lib/common"
	"xd/lib/version"
)

type Extension string

const PeerExchange = Extension("ut_pex")

func (ex Extension) String() string {
	return string(ex)
}

type ExtendedOptions struct {
	ID         uint8               `bencode:"-"`
	Version    string              `bencode:"v"`
	Extensions map[Extension]uint8 `bencode:"m"`
}

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

func (opts *ExtendedOptions) ToWireMessage() *common.WireMessage {
	b := new(bytes.Buffer)
	b.Write([]byte{opts.ID})
	if opts.ID == 0 {
		bencode.NewEncoder(b).Encode(opts)
	}
	return common.NewWireMessage(common.Extended, b.Bytes())
}

func New() *ExtendedOptions {
	return &ExtendedOptions{
		Version:    version.Version,
		Extensions: make(map[Extension]uint8),
	}
}

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
