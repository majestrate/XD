package swarm

import (
	"xd/lib/bittorrent/extensions"
)

type chatEvent struct {
	chat extensions.EphemChat
	from string
}
