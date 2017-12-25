// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package models

import (
	"sync"

	"github.com/majestrate/chihaya/config"
	"github.com/majestrate/chihaya/stats"
)

// PeerMap is a thread-safe map from PeerKeys to Peers. When PreferredSubnet is
// enabled, it is a thread-safe map of maps from MaskedIPs to Peerkeys to Peers.
type PeerMap struct {
	Peers   map[PeerKey]Peer
	Seeders bool `json:"seeders"`
	sync.RWMutex
}

// NewPeerMap initializes the map for a new PeerMap.
func NewPeerMap(seeders bool, cfg *config.Config) *PeerMap {
	pm := &PeerMap{
		Peers:   make(map[PeerKey]Peer),
		Seeders: seeders,
	}
	return pm
}

// Contains is true if a peer is contained with a PeerMap.
func (pm *PeerMap) Contains(pk PeerKey) bool {
	pm.RLock()
	defer pm.RUnlock()
	_, exists := pm.Peers[pk]
	return exists
}

// LookUp is a thread-safe read from a PeerMap.
func (pm *PeerMap) LookUp(pk PeerKey) (peer Peer, exists bool) {
	pm.RLock()
	defer pm.RUnlock()
	peer, exists = pm.Peers[pk]
	return
}

// Put is a thread-safe write to a PeerMap.
func (pm *PeerMap) Put(p Peer) {
	pm.Lock()
	defer pm.Unlock()
	pm.Peers[p.Key()] = p
}

// Delete is a thread-safe delete from a PeerMap.
func (pm *PeerMap) Delete(pk PeerKey) {
	pm.Lock()
	defer pm.Unlock()
	_, exists := pm.Peers[pk]
	if exists {
		delete(pm.Peers, pk)
	}
}

// Len returns the number of peers within a PeerMap.
func (pm *PeerMap) Len() int {
	pm.Lock()
	defer pm.Unlock()
	return len(pm.Peers)
}

// Purge iterates over all of the peers within a PeerMap and deletes them if
// they are older than the provided time.
func (pm *PeerMap) Purge(unixtime int64) {
	pm.Lock()
	defer pm.Unlock()
	for key, peer := range pm.Peers {
		if peer.LastAnnounce <= unixtime {
			delete(pm.Peers, key)
			if pm.Seeders {
				stats.RecordPeerEvent(stats.ReapedSeed)
			} else {
				stats.RecordPeerEvent(stats.ReapedLeech)
			}
		}
	}
}

func (pm *PeerMap) AppendPeers(peers PeerList, a *Announce, wanted int) (ls PeerList) {
	pm.Lock()
	defer pm.Unlock()
	for _, peer := range pm.Peers {
		if wanted > 0 {
			if peersEquivalent(a.Peer, &peer) {
				continue
			} else {
				ls = append(ls, peer)
				wanted--
			}
		} else {
			break
		}
	}
	return
}

// peersEquivalent checks if two peers represent the same entity.
func peersEquivalent(a, b *Peer) bool {
	return a.ID == b.ID || a.UserID != 0 && a.UserID == b.UserID
}
