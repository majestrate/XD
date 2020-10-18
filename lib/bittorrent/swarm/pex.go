package swarm

import (
	"net"
	"github.com/majestrate/XD/lib/network/i2p"
	"github.com/majestrate/XD/lib/sync"
)

// PEXSwarmState manages PeerExchange state on a bittorrent swarm
type PEXSwarmState struct {
	m sync.Map
}

func (p *PEXSwarmState) onNewPeer(addr net.Addr) {
	p.m.Store(addr.String(), true)
}

func (p *PEXSwarmState) onPeerDisconnected(addr net.Addr) {
	p.m.Store(addr.String(), false)
}

// PopDestHashList gets list of i2p destination hashes of currently active and disconnected peers
func (p *PEXSwarmState) PopDestHashLists() (connected, disconnected []byte) {
	p.m.Range(func(k, v interface{}) bool {
		addr := k.(string)
		active := v.(bool)
		h := i2p.I2PAddr(addr).Base32Addr()
		if active {
			connected = append(connected, h[:]...)
		} else {
			disconnected = append(disconnected, h[:]...)
			p.m.Delete(k)
		}
		return false
	})
	return
}
