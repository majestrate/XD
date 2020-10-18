package transmission

import "github.com/majestrate/XD/lib/bittorrent/swarm"

type Handler func(*swarm.Swarm, Args) Response
