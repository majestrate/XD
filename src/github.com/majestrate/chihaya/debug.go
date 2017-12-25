// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package chihaya

import (
	"flag"
	"net/http"
	"os"
	"runtime/pprof"

	_ "net/http/pprof"

	"xd/lib/log"
)

var (
	profile     string
	debugAddr   string
	profileFile *os.File
)

func init() {
	flag.StringVar(&profile, "profile", "", "if non-empty, path to write CPU profiling data")
	flag.StringVar(&debugAddr, "debug", "", "if non-empty, address to serve debug data")
}

func debugBoot() {
	var err error

	if debugAddr != "" {
		go func() {
			log.Infof("Starting debug HTTP on %s", debugAddr)
			e := http.ListenAndServe(debugAddr, nil)
			if e != nil {
				log.Fatalf("error: %s", e.Error())
			}
		}()
	}

	if profile != "" {
		profileFile, err = os.Create(profile)
		if err != nil {
			log.Fatalf("Failed to create profile file: %s\n", err)
		}

		pprof.StartCPUProfile(profileFile)
		log.Info("Started profiling")
	}
}

func debugShutdown() {
	if profileFile != nil {
		profileFile.Close()
		pprof.StopCPUProfile()
		log.Info("Stopped profiling")
	}
}
