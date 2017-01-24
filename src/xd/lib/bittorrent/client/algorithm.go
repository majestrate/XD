package client

import (
	"xd/lib/bittorrent"
	"xd/lib/common"
)

type Algorithm interface {
	// get next piece request given remote bitfiled
	Next(id common.PeerID, remote *bittorrent.Bitfield) *bittorrent.PieceRequest

	// are we done downloading ?
	Done() bool

	// should peer be choked right now ?
	Choke(id common.PeerID) bool
}
