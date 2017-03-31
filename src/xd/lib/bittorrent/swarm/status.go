package swarm

import (
	"bytes"
	"strings"
	"xd/lib/common"
)

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
	ID   common.PeerID
	Addr string
}

func (p *PeerConnStats) Less(o *PeerConnStats) bool {
	return bytes.Compare(p.ID[:], o.ID[:]) < 0
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
	return strings.Compare(l[i].Name, l[j].Name) < 0
}

func (l *TorrentStatusList) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}
