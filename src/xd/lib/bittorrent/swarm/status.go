package swarm

import "xd/lib/util"

type TorrentPeers []*PeerConnStats

func (p TorrentPeers) Len() int {
	return len(p)
}

func (p TorrentPeers) Less(i, j int) bool {
	return p[i].Less(p[j])
}

func (p *TorrentPeers) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}

// connection statistics
type PeerConnStats struct {
	TX   float32
	RX   float32
	ID   string
	Addr string
}

func (p *PeerConnStats) Less(o *PeerConnStats) bool {
	return util.StringCompare(p.ID, o.ID) < 0
}

type TorrentState string

const Seeding = TorrentState("seeding")
const Stopped = TorrentState("stopped")
const Downloading = TorrentState("downloading")

func (t TorrentState) String() string {
	return string(t)
}

// immutable status of torrent
type TorrentStatus struct {
	Peers    TorrentPeers
	Name     string
	State    TorrentState
	Infohash string
}

type TorrentStatusList []TorrentStatus

func (l TorrentStatusList) Len() int {
	return len(l)
}
func (l TorrentStatusList) Less(i, j int) bool {
	return util.StringCompare(l[i].Name, l[j].Name) < 0
}

func (l *TorrentStatusList) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}
