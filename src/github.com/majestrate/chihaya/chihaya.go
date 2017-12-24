// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package chihaya implements the ability to boot the Chihaya BitTorrent
// tracker with your own imports that can dynamically register additional
// functionality.
package chihaya

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"xd/lib/log"

	"github.com/majestrate/chihaya/api"
	"github.com/majestrate/chihaya/config"
	"github.com/majestrate/chihaya/http"
	"github.com/majestrate/chihaya/stats"
	"github.com/majestrate/chihaya/tracker"

	// noop tracker backend
	_ "github.com/majestrate/chihaya/backend/noop"
)

var (
	maxProcs   int
	configPath string
)

func init() {
	flag.IntVar(&maxProcs, "maxprocs", runtime.NumCPU(), "maximum parallel threads")
	flag.StringVar(&configPath, "config", "", "path to the configuration file")
}

type server interface {
	Serve()
	Stop()
}

// Boot starts Chihaya. By exporting this function, anyone can import their own
// custom drivers into their own package main and then call chihaya.Boot.
func Boot() {
	log.SetLevel("debug")
	flag.Parse()

	runtime.GOMAXPROCS(maxProcs)
	log.Infof("Set max threads to %d", maxProcs)

	debugBoot()
	defer debugShutdown()

	cfg, err := config.Open(configPath)
	if err != nil {
		log.Fatalf("Failed to parse configuration file: %s\n", err)
	}

	if cfg == &config.DefaultConfig {
		log.Info("Using default config")
	} else {
		log.Infof("Loaded config file: %s", configPath)
	}

	stats.DefaultStats = stats.New(cfg.StatsConfig)

	tkr, err := tracker.New(cfg)
	if err != nil {
		log.Fatalf("New: %s", err)
	}

	var servers []server

	if cfg.APIConfig.ListenAddr != "" {
		servers = append(servers, api.NewServer(cfg, tkr))
	}
	if cfg.Tor.Enabled {
		s := cfg.Tor.CreateSession()
		log.Info("opening tor session")
		err = s.Open()
		if err == nil {
			log.Info("opened tor session")
			err = s.SaveKey(cfg.Tor.Privkey)
			if err == nil {
				servers = append(servers, http.NewServer(s, cfg, tkr))
			}
		}
		if err != nil {
			log.Fatalf("failed: %s", err)
		}
	}
	if cfg.I2P.Enabled {
		s := cfg.I2P.CreateSession()
		log.Info("opening i2p session")
		err = s.Open()
		if err == nil {
			log.Info("opened i2p session")
			err = s.SaveKey(cfg.I2P.Keyfile)
			if err == nil {
				servers = append(servers, http.NewServer(s, cfg, tkr))
			}
		}
		if err != nil {
			log.Fatalf("failed: %s", err)
		}
	}

	var wg sync.WaitGroup
	for _, srv := range servers {
		wg.Add(1)
		// If you don't explicitly pass the server, every goroutine captures the
		// last server in the list.
		go func(srv server) {
			srv.Serve()
			wg.Done()
		}(srv)
	}

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		wg.Wait()
		signal.Stop(shutdown)
		close(shutdown)
	}()

	<-shutdown
	log.Info("Shutting down...")

	for _, srv := range servers {
		srv.Stop()
	}

	<-shutdown

	if err := tkr.Close(); err != nil {
		log.Errorf("Failed to shut down tracker cleanly: %s", err.Error())
	}
}
