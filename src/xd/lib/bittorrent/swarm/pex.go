package swarm

import (
	"net"
	"sync"
	"xd/lib/network/i2p"
)


// PEXSwarmState manages PeerExchange state on a bittorrent swarm
type PEXSwarmState struct {
	active map[string]bool
	access sync.Mutex
}

// Create a new PEXSwarmState
func NewPEXSwarmState() *PEXSwarmState {

	return &PEXSwarmState{
		active: make(map[string]bool),
	}
}

func (p *PEXSwarmState) onNewPeer(addr net.Addr) {
	p.access.Lock()
	p.active[addr.String()] = true
	p.access.Unlock()
}

func (p *PEXSwarmState) onPeerDisconnected(addr net.Addr) {
	p.access.Lock()
	p.active[addr.String()] = false
	p.access.Unlock()

}

// PopDestHashList gets list of i2p destination hashes of currently active and disconnected peers
func (p *PEXSwarmState) PopDestHashLists() (connected, disconnected []byte) {
	p.access.Lock()
	var remove []string
	for addr, active := range p.active {
		h := i2p.I2PAddr(addr).Base32Addr()
		if active {
			connected = append(connected, h[:]...)
		} else {
			disconnected = append(disconnected, h[:]...)
			remove = append(remove, addr)
		}
	}
	// clean up stale
	for _, addr := range remove {
		delete(p.active, addr)
	}
	p.access.Unlock()
	return
}
