package transmission

import "xd/lib/bittorrent/swarm"

type Handler func(*swarm.Swarm, Args) Response
