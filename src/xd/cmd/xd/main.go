package main

import (
	"xd/lib/bittorrent/swarm"
	"xd/lib/config"
	"xd/lib/log"
)

func main() {
	done := make(chan error)
	conf := new(config.Config)
	fname := "torrents.ini"
	err := conf.Load(fname)
	if err != nil {
		log.Errorf("failed to config %s", err)
		return
	}
	log.Info("loaded config")

	st := conf.Storage.CreateStorage()

	sw := swarm.NewSwarm(st)
	go func() {
		err := sw.AddTorrents()
		if err == nil {
			log.Info("running swarm")
			// run swarm
			err = sw.Run()
			if err != nil {
				log.Errorf("error in swarm runner: %s", err.Error())
			}
		} else {
			log.Errorf("failed to add all torrents: %s", err.Error())
		}
		done <- err
	}()
	
	
	net := conf.I2P.CreateSession()
	log.Info("opening i2p session")
	err = net.Open()
	if err != nil {
		log.Fatalf("failed to open i2p session: %s", err.Error())
	}
	log.Infof("i2p session made, we are %s", net.B32Addr())
	sw.SetNetwork(net)
	err = <- done
	close(done)
	// close network because we are done
	log.Info("closing i2p network connection")
	net.Close()
	log.Info("exited")
}
