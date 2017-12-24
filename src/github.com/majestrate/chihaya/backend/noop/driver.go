// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package noop implements a Chihaya backend storage driver as a no-op. This is
// useful for running Chihaya as a public tracker.
package noop

import (
	"github.com/majestrate/chihaya/backend"
	"github.com/majestrate/chihaya/config"
	"github.com/majestrate/chihaya/tracker/models"
)

type driver struct{}

// NoOp is a backend driver for Chihaya that does nothing. This is used by
// public trackers.
type NoOp struct{}

// New returns a new Chihaya backend driver that does nothing.
func (d *driver) New(cfg *config.DriverConfig) (backend.Conn, error) {
	return &NoOp{}, nil
}

// Close returns nil.
func (n *NoOp) Close() error {
	return nil
}

// Ping returns nil.
func (n *NoOp) Ping() error {
	return nil
}

// RecordAnnounce returns nil.
func (n *NoOp) RecordAnnounce(delta *models.AnnounceDelta) error {
	return nil
}

func (n *NoOp) DeleteTorrent(t *models.Torrent) error {
	return nil
}

func (n *NoOp) AddTorrent(t *models.Torrent) error {
	return nil
}

func (n *NoOp) DeleteUser(u *models.User) error {
	return nil
}

func (n *NoOp) AddUser(u *models.User) error {
	return nil
}

func (n *NoOp) GetTorrentByInfoHash(infohash string) (*models.Torrent, error) {
	return nil, nil
}

func (n *NoOp) GetUserByPassKey(key string) (*models.User, error) {
	return nil, nil
}

// LoadTorrents fetches and returns the specified torrents.
func (n *NoOp) LoadTorrents(ids []uint64) ([]*models.Torrent, error) {
	return nil, nil
}

// LoadUsers fetches and returns the specified users.
func (n *NoOp) LoadUsers(ids []uint64) ([]*models.User, error) {
	return nil, nil
}

// Init registers the noop driver as a backend for Chihaya.
func init() {
	backend.Register("noop", &driver{})
}
