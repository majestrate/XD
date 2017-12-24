// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"fmt"
	"io"
	"net/http"

	"strings"

	"github.com/majestrate/chihaya/stats"
	"github.com/majestrate/chihaya/tracker/models"
)

func handleTorrentError(err error, w *Writer) (int, error) {
	if err == nil {
		return http.StatusOK, nil
	} else if models.IsPublicError(err) {
		w.WriteError(err)
		stats.RecordEvent(stats.ClientError)
		return http.StatusOK, nil
	}

	return http.StatusInternalServerError, err
}

func (s *Server) serveScrapeAnon(w http.ResponseWriter, r *http.Request) (int, error) {
	return s.serveScrape(w, r, "")
}

func (s *Server) serveAnnounceAnon(w http.ResponseWriter, r *http.Request) (int, error) {
	return s.serveAnnounce(w, r, "")
}

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request, userid string) (int, error) {
	writer := &Writer{w, s.session}
	ann, err := s.newAnnounce(r, userid)
	if err != nil {
		return handleTorrentError(err, writer)
	}

	return handleTorrentError(s.tracker.HandleAnnounce(ann, writer), writer)
}

func (s *Server) serveScrape(w http.ResponseWriter, r *http.Request, userid string) (int, error) {
	writer := &Writer{w, s.session}
	scrape, err := s.newScrape(r, userid)
	if err != nil {
		return handleTorrentError(err, writer)
	}

	return handleTorrentError(s.tracker.HandleScrape(scrape, writer), writer)
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) (int, error) {
	addr := s.session.B32Addr()
	proto := "http"
	if strings.HasSuffix(addr, ".onion") {
		proto += "s"
	}
	txt := fmt.Sprintf("bittorrent open tracker announce url %s://%s/announce\n", addr)
	_, err := io.WriteString(w, txt)
	txt = fmt.Sprintf("to use:\n\nmktorrent -a %s://%s/announce somedirectory\n", addr)
	_, err = io.WriteString(w, txt)
	return http.StatusOK, err
}
