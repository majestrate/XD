package swarm

type RPC struct {
	sw *Swarm
}

func (r *RPC) TorrentStatus(num *int, status *TorrentStatus) error {
	return nil
}
