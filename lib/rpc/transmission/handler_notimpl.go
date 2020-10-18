package transmission

import (
	"github.com/majestrate/XD/lib/bittorrent/swarm"
)

func NotImplemented(sw *swarm.Swarm, args Args) (resp Response) {
	resp.Result = notImplemented
	resp.Args = make(Args)
	return
}
