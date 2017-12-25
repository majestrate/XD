// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"github.com/majestrate/chihaya/stats"
	"github.com/majestrate/chihaya/tracker/models"
)

// HandleAnnounce encapsulates all of the logic of handling a BitTorrent
// client's Announce without being coupled to any transport protocol.
func (tkr *Tracker) HandleAnnounce(ann *models.Announce, w Writer) (err error) {
	if tkr.Config.ClientWhitelistEnabled {
		if err = tkr.ClientApproved(ann.ClientID()); err != nil {
			return err
		}
	}

	var user *models.User
	if tkr.Config.PrivateEnabled {
		if user, err = tkr.FindUser(ann.Passkey); err != nil {
			return err
		}
	}

	torrent, err := tkr.FindTorrent(ann.Infohash)

	if err == models.ErrTorrentDNE && tkr.Config.CreateOnAnnounce {
		torrent = &models.Torrent{
			Infohash: ann.Infohash,
			Seeders:  models.NewPeerMap(true, tkr.Config),
			Leechers: models.NewPeerMap(false, tkr.Config),
		}

		tkr.PutTorrent(torrent)
		stats.RecordEvent(stats.NewTorrent)
	} else if err != nil {
		return err
	}

	ann.BuildPeer(user, torrent)
	var delta *models.AnnounceDelta

	if tkr.Config.PrivateEnabled {
		delta = newAnnounceDelta(ann, torrent)
	}

	created, err := tkr.updateSwarm(ann)
	if err != nil {
		return err
	}

	snatched, err := tkr.handleEvent(ann)
	if err != nil {
		return err
	}

	if tkr.Config.PrivateEnabled {
		delta.Created = created
		delta.Snatched = snatched
		if err = tkr.Backend.RecordAnnounce(delta); err != nil {
			return err
		}
	} else if tkr.Config.PurgeInactiveTorrents && torrent.PeerCount() == 0 {
		// Rather than deleting the torrent explicitly, let the tracker driver delete torrents
		// ensure there are no race conditions.
		tkr.PurgeInactiveTorrent(torrent.Infohash)
		stats.RecordEvent(stats.DeletedTorrent)
	}

	stats.RecordEvent(stats.Announce)
	return w.WriteAnnounce(newAnnounceResponse(ann))
}

// Builds a partially populated AnnounceDelta, without the Snatched and Created
// fields set.
func newAnnounceDelta(ann *models.Announce, t *models.Torrent) *models.AnnounceDelta {
	var oldUp, oldDown, rawDeltaUp, rawDeltaDown uint64

	switch {
	case t.Seeders.Contains(ann.Peer.Key()):
		oldPeer, _ := t.Seeders.LookUp(ann.Peer.Key())
		oldUp = oldPeer.Uploaded
		oldDown = oldPeer.Downloaded
	case t.Leechers.Contains(ann.Peer.Key()):
		oldPeer, _ := t.Leechers.LookUp(ann.Peer.Key())
		oldUp = oldPeer.Uploaded
		oldDown = oldPeer.Downloaded
	}

	// Restarting a torrent may cause a delta to be negative.
	if ann.Peer.Uploaded > oldUp {
		rawDeltaUp = ann.Peer.Uploaded - oldUp
	}
	if ann.Peer.Downloaded > oldDown {
		rawDeltaDown = ann.Peer.Downloaded - oldDown
	}

	uploaded := uint64(float64(rawDeltaUp) * ann.User.UpMultiplier * ann.Torrent.UpMultiplier)
	downloaded := uint64(float64(rawDeltaDown) * ann.User.DownMultiplier * ann.Torrent.DownMultiplier)

	if ann.Config.FreeleechEnabled {
		downloaded = 0
	}

	return &models.AnnounceDelta{
		Peer:    ann.Peer,
		Torrent: ann.Torrent,
		User:    ann.User,

		Uploaded:      uploaded,
		RawUploaded:   rawDeltaUp,
		Downloaded:    downloaded,
		RawDownloaded: rawDeltaDown,
	}
}

// updateSwarm handles the changes to a torrent's swarm given an announce.
func (tkr *Tracker) updateSwarm(ann *models.Announce) (created bool, err error) {
	tkr.TouchTorrent(ann.Torrent.Infohash)
	created, err = tkr.updatePeer(ann, ann.Peer)
	return
}

func (tkr *Tracker) updatePeer(ann *models.Announce, peer *models.Peer) (created bool, err error) {
	p, t := ann.Peer, ann.Torrent

	switch {
	case t.Seeders.Contains(p.Key()):
		err = tkr.PutSeeder(t.Infohash, p)
		if err != nil {
			return
		}

	case t.Leechers.Contains(p.Key()):
		err = tkr.PutLeecher(t.Infohash, p)
		if err != nil {
			return
		}

	default:
		if ann.Left == 0 {
			err = tkr.PutSeeder(t.Infohash, p)
			if err != nil {
				return
			}
			stats.RecordPeerEvent(stats.NewSeed)

		} else {
			err = tkr.PutLeecher(t.Infohash, p)
			if err != nil {
				return
			}
			stats.RecordPeerEvent(stats.NewLeech)
		}
		created = true
	}
	return
}

// handleEvent checks to see whether an announce has an event and if it does,
// properly handles that event.
func (tkr *Tracker) handleEvent(ann *models.Announce) (snatched bool, err error) {
	snatched, err = tkr.handlePeerEvent(ann, ann.Peer)
	if err == nil {
		err = tkr.IncrementTorrentSnatches(ann.Torrent.Infohash)
		if err == nil {
			ann.Torrent.Snatches++
			snatched = true
		}
	}
	return
}

func (tkr *Tracker) handlePeerEvent(ann *models.Announce, p *models.Peer) (snatched bool, err error) {
	p, t := ann.Peer, ann.Torrent

	switch {
	case ann.Event == "stopped" || ann.Event == "paused":
		// updateSwarm checks if the peer is active on the torrent,
		// so one of these branches must be followed.
		if t.Seeders.Contains(p.Key()) {
			err = tkr.DeleteSeeder(t.Infohash, p)
			if err != nil {
				return
			}
			stats.RecordPeerEvent(stats.DeletedSeed)

		} else if t.Leechers.Contains(p.Key()) {
			err = tkr.DeleteLeecher(t.Infohash, p)
			if err != nil {
				return
			}
			stats.RecordPeerEvent(stats.DeletedLeech)
		}

	case t.Leechers.Contains(p.Key()) && (ann.Event == "completed" || ann.Left == 0):
		// A leecher has completed or this is the first time we've seen them since
		// they've completed.
		err = tkr.leecherFinished(t, p)
		if err != nil {
			return
		}

		// Only mark as snatched if we receive the completed event.
		if ann.Event == "completed" {
			snatched = true
		}
	}

	return
}

// leecherFinished moves a peer from the leeching pool to the seeder pool.
func (tkr *Tracker) leecherFinished(t *models.Torrent, p *models.Peer) error {
	if err := tkr.DeleteLeecher(t.Infohash, p); err != nil {
		return err
	}

	if err := tkr.PutSeeder(t.Infohash, p); err != nil {
		return err
	}

	stats.RecordPeerEvent(stats.Completed)
	return nil
}

func newAnnounceResponse(ann *models.Announce) *models.AnnounceResponse {
	seedCount := ann.Torrent.Seeders.Len()
	leechCount := ann.Torrent.Leechers.Len()

	res := &models.AnnounceResponse{
		Announce:    ann,
		Complete:    seedCount,
		Incomplete:  leechCount,
		Interval:    int64(ann.Config.Announce.Duration.Seconds()),
		MinInterval: int64(ann.Config.MinAnnounce.Duration.Seconds()),
		Compact:     ann.Compact,
	}

	if ann.NumWant > 0 && ann.Event != "stopped" && ann.Event != "paused" {
		res.Peers = getPeers(ann)

		if len(res.Peers) == 0 {
			res.Peers = append(res.Peers, *ann.Peer)
		}
	}

	return res
}

// getPeers returns lists IPv4 and IPv6 peers on a given torrent sized according
// to the wanted parameter.
func getPeers(ann *models.Announce) (peers models.PeerList) {
	if ann.Left == 0 {
		// If they're seeding, give them only leechers.
		return ann.Torrent.Leechers.AppendPeers(peers, ann, ann.NumWant)
	}

	// If they're leeching, prioritize giving them seeders.
	peers = ann.Torrent.Seeders.AppendPeers(peers, ann, ann.NumWant)
	return ann.Torrent.Leechers.AppendPeers(peers, ann, ann.NumWant-len(peers))
}
