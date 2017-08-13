package extensions

import (
	"net"
	"xd/lib/common"
)

// PeerExchange is a BitTorrent Extension indicating we support PEX
const PeerExchange = Extension("i2p_pex")

func NewPEXMessage(peers []net.Addr) (msg *common.WireMessage) {
	return
}
