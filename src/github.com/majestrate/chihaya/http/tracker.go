// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"net/http"
	"net/url"
	"strconv"

	"net"
	"xd/lib/log"

	"github.com/majestrate/chihaya/tracker/models"
	"github.com/majestrate/chihaya/util"
)

// newAnnounce parses an HTTP request and generates a models.Announce.
func (s *Server) newAnnounce(r *http.Request, userID string) (*models.Announce, error) {

	q := r.URL.Query()

	event, _ := util.QueryParamString(q, "event")
	numWant := requestedPeerCount(q, s.config.NumWantFallback)

	infohash, exists := util.QueryParamString(q, "info_hash")
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	peerID, exists := util.QueryParamString(q, "peer_id")
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	port, exists := util.QueryParamUInt64(q, "port")
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	left, exists := util.QueryParamUInt64(q, "left")
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	addr, err := s.lookupAddr(r)
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	downloaded, exists := util.QueryParamUInt64(q, "downloaded")
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	uploaded, exists := util.QueryParamUInt64(q, "uploaded")
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	a := &models.Announce{
		Config:     s.config,
		Compact:    true,
		Downloaded: downloaded,
		Event:      event,
		Infohash:   infohash,
		Left:       left,
		NumWant:    numWant,
		Passkey:    userID,
		PeerID:     peerID,
		Uploaded:   uploaded,
	}
	a.Addr, _, err = net.SplitHostPort(addr.String())
	if err != nil {
		return nil, models.ErrMalformedRequest
	}
	a.Port = uint16(port)
	return a, nil
}

// newScrape parses an HTTP request and generates a models.Scrape.
func (s *Server) newScrape(r *http.Request, userID string) (*models.Scrape, error) {
	q := r.URL.Query()

	var infohashes []string
	if _, exists := q["info_hash"]; !exists {
		// There aren't any infohashes.
		return nil, models.ErrMalformedRequest
	}
	infohashes = q["info_hash"]
	return &models.Scrape{
		Config:     s.config,
		Passkey:    userID,
		Infohashes: infohashes,
	}, nil
}

// requestedPeerCount returns the wanted peer count or the provided fallback.
func requestedPeerCount(q url.Values, fallback int) int {
	if numWantStr, exists := util.QueryParamString(q, "numwant"); exists {
		numWant, err := strconv.Atoi(numWantStr)
		if err != nil {
			return fallback
		}
		return numWant
	}

	return fallback
}

// obtain the "real" address from the remote request
func (s *Server) lookupAddr(r *http.Request) (addr net.Addr, err error) {
	var a, p string
	a, p, err = net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		addr, err = s.session.Lookup(a, p)
	}
	if err != nil {
		log.Errorf("failed to lookup %s: %s", a, err)
	}
	return
}
