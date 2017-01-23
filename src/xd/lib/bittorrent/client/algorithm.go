package client

import (
	"xd/lib/bittorrent"
	"xd/lib/common"
)

type Algorithm interface {
	// get next piece request given remote and local bitfileds
	Next(id common.PeerID, remote, local *bittorrent.Bitfield) *bittorrent.PieceRequest

	// are we done downloading ?
	Done() bool

	// should peer be choked right now ?
	Choke(id common.PeerID) bool
}
