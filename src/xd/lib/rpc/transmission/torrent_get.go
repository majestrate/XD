package transmission

import (
	"time"
	"xd/lib/bittorrent/swarm"
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
}
