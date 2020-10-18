package transmission

import (
	"time"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
)

type tgResp map[string]interface{}

func (t *tgResp) Set(key string, val interface{}) {
	(*t)[key] = val
}

type tgFieldHandler func(string, *swarm.Torrent, *tgResp) error

func tgID(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, t.TID)
	return
}

func tgName(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, t.Name())
	return
}

func tgUploadRate(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, t.TX())
	return
}

func tgDownloadRate(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, t.RX())
	return
}

func tgDownloadDir(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, t.DownloadDir())
	return
}

func tgStatus(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	status := t.GetStatus()
	trStatus := tr_Status_Stopped
	switch status.State {
	case swarm.Downloading:
		trStatus = tr_Status_Download
	case swarm.Seeding:
		trStatus = tr_Status_Seed
	case swarm.Checking:
		trStatus = tr_Status_Check
	}
	resp.Set(f, trStatus)
	return
}

func tgZeroInt(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, 0)
	return
}

func tgFalse(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, false)
	return
}

func tgTrue(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, true)
	return
}

func tgZeroStr(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, "")
	return
}

func tgActivityDate(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	// TODO: implement
	resp.Set(f, time.Now().Unix())
	return
}

func tgAddedDate(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	resp.Set(f, t.AddedAt().Unix())
	return
}

func tgBwPrior(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	// TODO: implement
	resp.Set(f, tr_Pri_Norm)
	return
}

func tgComment(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	var comment string
	m := t.MetaInfo()
	if m != nil {
		comment = string(m.Comment)
	}
	resp.Set(f, comment)
	return
}

func tgBytesAvail(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	var avail int64
	m := t.MetaInfo()
	if m != nil {
		bf := t.Bitfield()
		if bf != nil {
			bf = bf.Copy()
			bf.Zero()
			t.VisitPeers(func(c *swarm.PeerConn) {
				pbf := c.Bitfield()
				if pbf != nil {
					bf.SelfOR(pbf)
				}
			})
			avail = int64(bf.CountSet())
			avail *= int64(m.Info.PieceLength)
		}
	}
	resp.Set(f, avail)
	return
}

type tgFile struct {
	Completed int64  `json:"bytesCompleted"`
	Length    int64  `json:"length"`
	Name      string `json:"name"`
}

func tgFiles(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	stats := t.GetStatus()
	files := make([]*tgFile, len(stats.Files))
	for idx := range stats.Files {
		files[idx] = &tgFile{
			Completed: stats.Files[idx].BytesCompleted(),
			Length:    stats.Files[idx].Length(),
			Name:      stats.Files[idx].Name(),
		}
	}
	resp.Set(f, files)
	return
}

type tgFileStat struct {
	Completed int64 `json:"bytesCompleted"`
	Wanted    bool  `json:"wanted"`
	Priority  int   `json:"priority"`
}

func tgFileStats(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	stats := t.GetStatus()
	files := make([]*tgFileStat, len(stats.Files))
	for idx := range stats.Files {
		files[idx] = &tgFileStat{
			Completed: stats.Files[idx].BytesCompleted(),
			Wanted:    true,
			Priority:  tr_Pri_Norm,
		}
	}
	resp.Set(f, files)
	return
}

type tgPeer struct {
	Addr            string  `json:"address"`
	ClientName      string  `json:"clientName"`
	UsChoked        bool    `json:"clientIsChoked"`
	UsInterested    bool    `json:"clientIsInterested"`
	Flag            string  `json:"flagStr"`
	DownloadingFrom bool    `json:"isDownloadingFrom"`
	Encrypted       bool    `json:"isEncrypted"`
	Inbound         bool    `json:"isIncoming"`
	Uploading       bool    `json:"isUploadingTo"`
	UTP             bool    `json:"isUTP"`
	ThemChoked      bool    `json:"peerIsChoked"`
	ThemInterested  bool    `json:"peerIsInterested"`
	Port            int     `json:"port"`
	Progress        float64 `json:"progress"`
	RX              int64   `json:"rateToClient"`
	TX              int64   `json:"rateToPeer"`
}

func tgPeers(f string, t *swarm.Torrent, resp *tgResp) (err error) {
	stats := t.GetStatus()
	peers := make([]*tgPeer, len(stats.Peers))
	for idx := range stats.Peers {
		peers[idx] = &tgPeer{
			Addr:            stats.Peers[idx].Addr,
			ClientName:      stats.Peers[idx].Client,
			UsChoked:        stats.Peers[idx].ThemChoking,
			UsInterested:    stats.Peers[idx].UsInterested,
			Flag:            "i2p",
			DownloadingFrom: stats.Peers[idx].Downloading,
			Inbound:         stats.Peers[idx].Inbound,
			Uploading:       stats.Peers[idx].Uploading,
			ThemChoked:      stats.Peers[idx].UsChoking,
			ThemInterested:  stats.Peers[idx].ThemInterested,
			Progress:        stats.Peers[idx].Bitfield.Progress(),
			RX:              int64(stats.Peers[idx].RX),
			TX:              int64(stats.Peers[idx].TX),
		}
	}
	return
}

var tgFieldHandlers = map[string]tgFieldHandler{
	"id":                tgID,
	"name":              tgName,
	"rateUpload":        tgUploadRate,
	"rateDownload":      tgDownloadRate,
	"downloadDir":       tgDownloadDir,
	"status":            tgStatus,
	"error":             tgZeroInt, // TODO
	"errorString":       tgZeroStr, // TODO
	"activityDate":      tgActivityDate,
	"addedDate":         tgAddedDate,
	"bandwidthPriority": tgBwPrior,
	"comment":           tgComment,
	"corruptEver":       tgZeroInt, // TODO
	"creator":           tgZeroStr, // TODO
	"dateCreated":       tgZeroInt, // TODO
	"desiredAvailable":  tgBytesAvail,
	"dowwloadLimit":     tgZeroInt, // TODO
	"downloadLimited":   tgFalse,   // TODO
	"doneDate":          tgZeroInt, // TODO
	"downloadedEver":    tgZeroInt, // TODO
	"eta":               tgZeroInt, // TODO
	"etaIdle":           tgZeroInt, // TODO
	"files":             tgFiles,
	"fileStats":         tgFileStats,
	"peers":             tgPeers,
}
