// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"bytes"
	"net/http"

	"github.com/majestrate/chihaya/tracker/models"
	"github.com/zeebo/bencode"
	"xd/lib/network"
)

// Writer implements the tracker.Writer interface for the HTTP protocol.
type Writer struct {
	http.ResponseWriter
	session network.Network
}

// WriteError writes a bencode dict with a failure reason.
func (w *Writer) WriteError(err error) error {
	bencoder := bencode.NewEncoder(w)
	w.Header().Set("Content-Type", "text/plain")
	return bencoder.Encode(map[string]interface{}{
		"failure reason": err.Error(),
	})
}

// WriteAnnounce writes a bencode dict representation of an AnnounceResponse.
func (w *Writer) WriteAnnounce(res *models.AnnounceResponse) error {
	dict := map[string]interface{}{
		"complete":     res.Complete,
		"incomplete":   res.Incomplete,
		"interval":     res.Interval,
		"min interval": res.MinInterval,
	}
	if res.Compact {
		dict["compact"] = 1
		dict["peers"] = compactPeers(w.session, res.Peers)
	} else {
		dict["compact"] = 0
		dict["peers"] = res.Peers
	}

	w.Header().Set("Content-Type", "text/plain")
	bencoder := bencode.NewEncoder(w)
	return bencoder.Encode(dict)
}

// WriteScrape writes a bencode dict representation of a ScrapeResponse.
func (w *Writer) WriteScrape(res *models.ScrapeResponse) error {
	dict := map[string]interface{}{
		"files": filesDict(res.Files),
	}

	w.Header().Set("Content-Type", "text/plain")
	bencoder := bencode.NewEncoder(w)
	return bencoder.Encode(dict)
}

func compactPeers(session network.Network, peers models.PeerList) []byte {
	var compactPeers bytes.Buffer
	for _, peer := range peers {
		compactPeers.Write(session.AddrToCompact(peer.Addr))
	}
	return compactPeers.Bytes()
}

func filesDict(torrents []*models.Torrent) map[string]interface{} {
	d := make(map[string]interface{})
	for _, torrent := range torrents {
		d[torrent.Infohash] = torrentDict(torrent)
	}
	return d
}

func torrentDict(torrent *models.Torrent) map[string]interface{} {
	return map[string]interface{}{
		"complete":   torrent.Seeders.Len(),
		"incomplete": torrent.Leechers.Len(),
		"downloaded": torrent.Snatches,
	}
}
