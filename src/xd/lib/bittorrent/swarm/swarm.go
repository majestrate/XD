package swarm

import (
	"xd/lib/network"
	"xd/lib/storage"
)

type Swarm struct {
	net network.Network
	storage storage.Torrent
}


func NewSwarm(storage storage.Torrent, net network.Network) *Swarm {
	return &Swarm{
		net: net,
		storage: storage,
	}
}
