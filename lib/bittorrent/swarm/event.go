package swarm

import (
	"github.com/majestrate/XD/lib/common"
)

// an event triggered when we get an inbound wire message from a peer we are connected with on this torrent asking for a piece
type pieceEvent struct {
	c *PeerConn
	r *common.PieceRequest
}
