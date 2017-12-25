// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package http implements a BitTorrent tracker over the HTTP protocol as per
// BEP 3.
package http

import (
	"net"
	"net/http"
	"time"

	"xd/lib/log"
	"xd/lib/network"

	"strings"

	"github.com/majestrate/chihaya/config"
	"github.com/majestrate/chihaya/stats"
	"github.com/majestrate/chihaya/tracker"
	"github.com/majestrate/chihaya/util"
)

// ResponseHandler is an HTTP handler that returns a status code.
type ResponseHandler func(http.ResponseWriter, *http.Request) (int, error)

// Server represents an HTTP serving torrent tracker.
type Server struct {
	session  network.Network
	config   *config.Config
	tracker  *tracker.Tracker
	server   *http.Server
	stopping bool
}

// makeHandler wraps our ResponseHandlers while timing requests, collecting,
// stats, logging, and handling errors.
func makeHandler(handler ResponseHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		httpCode, err := handler(w, r)
		duration := time.Since(start)

		var msg string
		if err != nil {
			msg = err.Error()
		} else if httpCode != http.StatusOK {
			msg = http.StatusText(httpCode)
		}

		if len(msg) > 0 {
			http.Error(w, msg, httpCode)
			stats.RecordEvent(stats.ErroredRequest)
		}

		if len(msg) > 0 {
			reqString := r.URL.Path + " " + r.RemoteAddr

			if len(msg) > 0 {
				log.Errorf("[HTTP - %9s] %s (%d - %s)", duration, reqString, httpCode, msg)
			} else {
				log.Infof("[HTTP - %9s] %s (%d)", duration, reqString, httpCode)
			}
		}

		stats.RecordEvent(stats.HandledRequest)
		stats.RecordTiming(stats.ResponseTime, duration)
	}
}

func (s *Server) serveAuthed(w http.ResponseWriter, r *http.Request) (int, error) {
	parts := strings.Split(r.URL.Path, "/")
	userid := parts[2]
	if strings.HasSuffix(r.URL.Path, "/announce") {
		return s.serveAnnounce(w, r, userid)
	} else if strings.HasSuffix(r.URL.Path, "/scrape") {
		return s.serveScrape(w, r, userid)
	} else {
		return http.StatusNotFound, nil
	}
}

// newMux returns a router with all the routes.
func newMux(s *Server) *http.ServeMux {
	mux := http.NewServeMux()

	if s.config.PrivateEnabled {
		util.GET(mux, "/users/", makeHandler(s.serveAuthed))
	} else {
		util.GET(mux, "/announce", makeHandler(s.serveAnnounceAnon))
		util.GET(mux, "/scrape", makeHandler(s.serveScrapeAnon))
	}
	util.GET(mux, "/", makeHandler(s.serveIndex))
	return mux
}

// connState is used by graceful in order to gracefully shutdown. It also
// keeps track of connection stats.
func (s *Server) connState(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		stats.RecordEvent(stats.AcceptedConnection)

	case http.StateClosed:
		stats.RecordEvent(stats.ClosedConnection)

	case http.StateHijacked:
		panic("connection impossibly hijacked")

	// Ignore the following cases.
	case http.StateActive, http.StateIdle:

	default:
		log.Errorf("Connection transitioned to unknown state %s (%d)", state, state)
	}
}

// Serve runs an HTTP server, blocking until the server has shut down.
func (s *Server) Serve() {
	log.Infof("Serving on %s", s.session.B32Addr())
	mux := newMux(s)
	serv := &http.Server{
		Handler:      mux,
		ReadTimeout:  s.config.HTTPConfig.ReadTimeout.Duration,
		WriteTimeout: s.config.HTTPConfig.WriteTimeout.Duration,
	}
	// disable keepalive
	serv.SetKeepAlivesEnabled(true)
	err := serv.Serve(s.session)

	if err != nil {
		log.Error(err.Error())
	}
	log.Info("HTTP server shut down cleanly")
}

// Stop cleanly shuts down the server.
func (s *Server) Stop() {
	if !s.stopping {
		s.stopping = true
		s.session.Close()
	}
}

// NewServer returns a new HTTP server for a given configuration and tracker.
func NewServer(session network.Network, cfg *config.Config, tkr *tracker.Tracker) *Server {
	log.Infof("New server at %s", session.B32Addr())
	return &Server{
		session: session,
		config:  cfg,
		tracker: tkr,
	}
}
