package swarm

// immutable status of torrent
type TorrentStatus struct {
	Peers []*PeerConnStats
}
