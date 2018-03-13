package transmission

import (
	"xd/lib/bittorrent/swarm"
)

func NotImplemented(sw *swarm.Swarm, args Args) (resp Response) {
	resp.Result = notImplemented
	resp.Args = make(Args)
	return
}
