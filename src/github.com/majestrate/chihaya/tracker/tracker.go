// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package tracker provides a generic interface for manipulating a
// BitTorrent tracker's fast-moving data.
package tracker

import (
	"time"

	"xd/lib/log"

	"github.com/majestrate/chihaya/backend"
	"github.com/majestrate/chihaya/config"
	"github.com/majestrate/chihaya/tracker/models"
)

// Tracker represents the logic necessary to service BitTorrent announces,
// independently of the underlying data transports used.
type Tracker struct {
	Config  *config.Config
	Backend backend.Conn
	Cache   *Storage
}

// New creates a new Tracker, and opens any necessary connections.
// Maintenance routines are automatically spawned in the background.
func New(cfg *config.Config) (*Tracker, error) {
	bc, err := backend.Open(&cfg.DriverConfig)
	if err != nil {
		return nil, err
	}

	tkr := &Tracker{
		Config:  cfg,
		Backend: bc,
		Cache:   NewStorage(cfg),
	}

	go tkr.purgeInactivePeers(
		cfg.PurgeInactiveTorrents,
		time.Duration(float64(cfg.MinAnnounce.Duration)*cfg.ReapRatio),
		cfg.ReapInterval.Duration,
	)

	if cfg.ClientWhitelistEnabled {
		tkr.LoadApprovedClients(cfg.ClientWhitelist)
	}

	return tkr, nil
}

// check if a peerID is approved
func (tkr *Tracker) ClientApproved(peerID string) (err error) {
	err = tkr.Cache.ClientApproved(peerID)
	return
}

// find user given passkey
func (tkr *Tracker) FindUser(passkey string) (u *models.User, err error) {
	// check cache first
	u, err = tkr.Cache.FindUser(passkey)
	if err == models.ErrUserDNE {
		if tkr.Config.PrivateEnabled {
			u, err = tkr.Backend.GetUserByPassKey(passkey)
		}
		if err == nil {
			// yey we got it
			// cache it
			tkr.Cache.PutUser(u)
		}
	}
	return
}

// find a torrent, checks cache then looks it up
func (tkr *Tracker) FindTorrent(infohash string) (t *models.Torrent, err error) {
	t, err = tkr.Cache.FindTorrent(infohash)
	if err == models.ErrTorrentDNE {
		// not in cache
		// let's check if it's registered
		if tkr.Config.PrivateEnabled {
			t, err = tkr.Backend.GetTorrentByInfoHash(infohash)
			if err == nil {
				t.Seeders = models.NewPeerMap(true, tkr.Config)
				t.Leechers = models.NewPeerMap(false, tkr.Config)
				// let's put it in the cache
				tkr.Cache.PutTorrent(t)
			}
		}
	}
	return
}

// put a torrent into the database
func (tkr *Tracker) PutTorrent(torrent *models.Torrent) (err error) {
	if tkr.Config.PrivateEnabled {
		err = tkr.Backend.AddTorrent(torrent)
	}
	tkr.Cache.PutTorrent(torrent)
	return
}

// purge an inactive torrent from the cache
func (tkr *Tracker) PurgeInactiveTorrent(infohash string) {
	tkr.Cache.PurgeInactiveTorrent(infohash)
}

// touch a torrent in cache
func (tkr *Tracker) TouchTorrent(infohash string) (err error) {
	err = tkr.Cache.TouchTorrent(infohash)
	return
}

// put a seeder into the cache
func (tkr *Tracker) PutSeeder(infohash string, p *models.Peer) (err error) {
	err = tkr.Cache.PutSeeder(infohash, p)
	return
}

// put a leecher into the cache
func (tkr *Tracker) PutLeecher(infohash string, p *models.Peer) (err error) {
	err = tkr.Cache.PutLeecher(infohash, p)
	return
}

// increment snatches for a torrent with an infohash
func (tkr *Tracker) IncrementTorrentSnatches(infohash string) (err error) {
	err = tkr.Cache.IncrementTorrentSnatches(infohash)
	return
}

// delete seeder from cache
func (tkr *Tracker) DeleteSeeder(infohash string, p *models.Peer) (err error) {
	err = tkr.Cache.DeleteSeeder(infohash, p)
	return
}

// delete leecher from cache
func (tkr *Tracker) DeleteLeecher(infohash string, p *models.Peer) (err error) {
	err = tkr.Cache.DeleteLeecher(infohash, p)
	return
}

// delete torrent from database
func (tkr *Tracker) DeleteTorrent(infohash string) error {
	t, err := tkr.FindTorrent(infohash)
	if err == nil && tkr.Config.PrivateEnabled {
		// remove from backend
		err = tkr.Backend.DeleteTorrent(t)
	}

	// remove from cache
	tkr.Cache.DeleteTorrent(infohash)
	return err
}

// put new user into database
// populate the user model with info
func (tkr *Tracker) RegisterUser(u *models.User) (user *models.User, err error) {
	err = tkr.Backend.AddUser(u)
	if err == nil {
		// user added gud
		var added []*models.User
		// let's get the full info we want from the backend
		added, err = tkr.Backend.LoadUsers([]uint64{u.ID})
		if err == nil {
			// user info retrieved from backend
			user = added[0]
			// put the user in the cache
			tkr.Cache.PutUser(user)
		}
	}
	return
}

func (tkr *Tracker) DeleteUser(passkey string) (err error) {
	var u *models.User
	u, err = tkr.Backend.GetUserByPassKey(passkey)
	if err == nil {
		// remove from backend
		err = tkr.Backend.DeleteUser(u)
		// remove from cache too
		tkr.Cache.DeleteUser(u.Passkey)
	}
	return
}

// Close gracefully shutdowns a Tracker by closing any database connections.
func (tkr *Tracker) Close() error {
	return tkr.Backend.Close()
}

// LoadApprovedClients loads a list of client IDs into the tracker's storage.
func (tkr *Tracker) LoadApprovedClients(clients []string) {
	for _, client := range clients {
		tkr.Cache.PutClient(client)
	}
}

// Writer serializes a tracker's responses, and is implemented for each
// response transport used by the tracker. Only one of these may be called
// per request, and only once.
//
// Note, data passed into any of these functions will not contain sensitive
// information, so it may be passed back the client freely.
type Writer interface {
	WriteError(err error) error
	WriteAnnounce(*models.AnnounceResponse) error
	WriteScrape(*models.ScrapeResponse) error
}

// purgeInactivePeers periodically walks the torrent database and removes
// peers that haven't announced recently.
func (tkr *Tracker) purgeInactivePeers(purgeEmptyTorrents bool, threshold, interval time.Duration) {
	for _ = range time.NewTicker(interval).C {
		before := time.Now().Add(-threshold)
		log.Infof("Purging peers with no announces since %s", before)
		// clear cache
		err := tkr.Cache.PurgeInactivePeers(purgeEmptyTorrents, before)
		if err != nil {
			log.Errorf("Error purging torrents: %s", err)
		}
	}
}
