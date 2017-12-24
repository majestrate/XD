// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package api implements a RESTful HTTP JSON API server for a BitTorrent
// tracker.
package api

import (
	"net"
	"net/http"
	"time"

	"xd/lib/log"

	"github.com/majestrate/chihaya/config"
	"github.com/majestrate/chihaya/stats"
	"github.com/majestrate/chihaya/tracker"
	"github.com/majestrate/chihaya/util"
)

// Server represents an API server for a torrent tracker.
type Server struct {
	config   *config.Config
	tracker  *tracker.Tracker
	server   *http.Server
	stopping bool
}

// NewServer returns a new API server for a given configuration and tracker
// instance.
func NewServer(cfg *config.Config, tkr *tracker.Tracker) *Server {
	return &Server{
		config:  cfg,
		tracker: tkr,
	}
}

// Stop cleanly shuts down the server.
func (s *Server) Stop() {
	if !s.stopping {
		s.server.Close()
	}
}

// Serve runs an API server, blocking until the server has shut down.
func (s *Server) Serve() {
	log.Infof("Starting API on %s", s.config.APIConfig.ListenAddr)

	if s.config.APIConfig.ListenLimit != 0 {
		log.Infof("Limiting connections to %d", s.config.APIConfig.ListenLimit)
	}

	s.server = &http.Server{
		Addr:         s.config.APIConfig.ListenAddr,
		Handler:      newMux(s),
		ReadTimeout:  s.config.APIConfig.ReadTimeout.Duration,
		WriteTimeout: s.config.APIConfig.WriteTimeout.Duration,
	}

	if err := s.server.ListenAndServe(); err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			log.Errorf("Failed to gracefully run API server: %s", err.Error())
			return
		}
	}

	log.Info("API server shut down cleanly")
}

// newRouter returns a router with all the routes.
func newMux(s *Server) http.Handler {
	mux := http.NewServeMux()

	if s.config.PrivateEnabled {
		// put a user with a passkey into the database
		//util.PUT(mux, "/users/", makeHandler(s.putUser))
		// remove a user with a passkey from the database
		//util.DELETE(mux, "/users/", makeHandler(s.delUser))

		/*
		   // get category list
		   r.GET("/list/cats", makeHandler(s.listCategories))
		   // get page for category
		   r.GET("/list/cat/:id", makeHandler(s.listCategory))
		   // get search results for tag
		   r.GET("/list/tag/:tag", makeHandler(s.listTag))
		*/
	}

	if s.config.ClientWhitelistEnabled {
		util.GET(mux, "/clients/", makeHandler(s.getClient))
		//util.PUT(mux, "/clients/", makeHandler(s.putClient))
		//util.DELETE(mux, "/clients/", makeHandler(s.delClient))
	}

	// get top torrent swarms
	util.GET(mux, "/top/", makeHandler(s.getTopSwarms))
	// get torrent info
	util.GET(mux, "/torrents/", makeHandler(s.getTorrent))
	// add torrent to backend
	//util.PUT(mux, "/torrents/", makeHandler(s.putTorrent))
	// delete torrent from backend
	//util.DELETE(mux, "/torrents/", makeHandler(s.delTorrent))
	// check if backend is alive
	util.GET(mux, "/check", makeHandler(s.check))
	// get stats
	util.GET(mux, "/stats", makeHandler(s.stats))
	// dump all info
	util.GET(mux, "/dump", makeHandler(s.dumpAll))
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

// ResponseHandler is an HTTP handler that returns a status code.
type ResponseHandler func(http.ResponseWriter, *http.Request) (int, error)

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
			log.Errorf("[API - %9s] %s (%d - %s)", duration, reqString, httpCode, msg)
		}

		stats.RecordEvent(stats.HandledRequest)
		stats.RecordTiming(stats.ResponseTime, duration)
	}
}
