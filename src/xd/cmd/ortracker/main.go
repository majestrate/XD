package main

import (
	"os"

	"xd/lib/log"
	"xd/lib/ortracker/config"
)

func main() {
	cfg_fname := "ortracker.ini"
	cfg := new(config.Config)
	if len(os.Args) == 2 {
		cfg_fname = os.Args[1]
	}
	log.SetLevel("debug")
	err := cfg.Load(cfg_fname)
	if err != nil {
		log.Errorf("failed to load config: %s", err.Error())
		return
	}
	s1 := cfg.Tor.CreateSession()
	s2 := cfg.Tor.CreateSession()
	err = s1.Open()
	if err == nil {
		log.Infof("session open as %s", s1.B32Addr())
		if err == nil {
			a := "ev7fnjzjdbtu3miq.onion"
			c, e := s1.Lookup(a, "80")
			if e == nil {
				log.Infof("found %s", c)
			} else {
				log.Errorf("error: %s", e)
			}
		}
	}
	if err != nil {
		log.Errorf("error: %s", err.Error())
	}
	if s1 != nil {
		s1.Close()
	}
	if s2 != nil {
		s2.Close()
	}
}
