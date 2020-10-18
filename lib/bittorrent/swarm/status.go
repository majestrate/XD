package swarm

import (
	"fmt"
	"github.com/majestrate/XD/lib/bittorrent"
	"github.com/majestrate/XD/lib/metainfo"
	"github.com/majestrate/XD/lib/util"
)

type TorrentFileInfo struct {
	FileInfo metainfo.FileInfo
	Progress float64
}

func (i TorrentFileInfo) Length() int64 {
	return int64(i.FileInfo.Length)
}

func (i TorrentFileInfo) Name() string {
	return i.FileInfo.Path.FilePath("")
}

func (i TorrentFileInfo) BytesCompleted() int64 {
	return int64(float64(i.FileInfo.Length) * i.Progress)
}

type TorrentPeers []*PeerConnStats

func (p TorrentPeers) RX() (rx float64) {
	for idx := range p {
		if p[idx] != nil {
			rx += p[idx].RX
		}
	}
	return
}

func (p TorrentPeers) TX() (tx float64) {
	for idx := range p {
		if p[idx] != nil {
			tx += p[idx].TX
		}
	}
	return
}

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
	TX             float64
	RX             float64
	ID             string
	Client         string
	Addr           string
	UsInterested   bool
	UsChoking      bool
	ThemInterested bool
	ThemChoking    bool
	Downloading    bool
	Inbound        bool
	Uploading      bool
	Bitfield       bittorrent.Bitfield
}

func (p *PeerConnStats) Less(o *PeerConnStats) bool {
	return util.StringCompare(p.ID, o.ID) < 0
}

type TorrentState string

const Seeding = TorrentState("seeding")
const Checking = TorrentState("checking")
const Stopped = TorrentState("stopped")
const Downloading = TorrentState("downloading")

func (t TorrentState) String() string {
	return string(t)
}

// immutable status of torrent
type TorrentStatus struct {
	Files    []TorrentFileInfo
	Peers    TorrentPeers
	Us       PeerConnStats
	Name     string
	State    TorrentState
	Infohash string
	Progress float64
	TX       uint64
	RX       uint64
}

func (t TorrentStatus) Ratio() (r float64) {
	r = util.Ratio(float64(t.TX), float64(t.RX))
	return
}

type TorrentStatusList []TorrentStatus

func (l TorrentStatusList) TX() (tx float64) {
	for idx := range l {
		tx += l[idx].Peers.TX()
	}
	return
}

func (l TorrentStatusList) Ratio() (r float64) {
	var tx, rx uint64
	for idx := range l {
		tx += l[idx].TX
		rx += l[idx].RX
	}
	r = util.Ratio(float64(tx), float64(rx))
	return
}

func (l TorrentStatusList) RX() (rx float64) {
	for idx := range l {
		rx += l[idx].Peers.RX()
	}
	return
}

func (l TorrentStatusList) Len() int {
	return len(l)
}
func (l TorrentStatusList) Less(i, j int) bool {
	return util.StringCompare(l[i].Name, l[j].Name) < 0
}

func (l *TorrentStatusList) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

// SwarmBandwidth is a string tuple for bandwith
type SwarmBandwidth struct {
	Upload   string
	Download string
}

func (sb SwarmBandwidth) String() string {
	return fmt.Sprintf("Upload: %s Download: %s", sb.Upload, sb.Download)
}

// infohash -> torrent status map
type SwarmStatus map[string]TorrentStatus

func (sw SwarmStatus) TotalSpeed() (tx, rx float64) {
	for ih := range sw {
		tx += sw[ih].Peers.TX()
		rx += sw[ih].Peers.RX()
	}
	return
}

func (sw SwarmStatus) Ratio() (r float64) {
	var tx, rx uint64
	for ih := range sw {
		tx += sw[ih].TX
		rx += sw[ih].RX
	}
	r = util.Ratio(float64(tx), float64(rx))
	return
}
