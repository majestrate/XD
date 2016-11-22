package main

import (
	"xd/lib/config"
	"xd/lib/log"
)

func main() {
	conf := new(config.Config)
	fname := "torrents.ini"
	err := conf.Load(fname)
	if err != nil {
		log.Errorf("failed to config %s", err)
		return
	}
	log.Info("loaded config")
	net := conf.I2P.CreateSession()
	log.Info("opening i2p session")
	err = net.Open()
	if err != nil {
		log.Fatalf("failed to open i2p session: %s", err.Error())
	}
	log.Infof("i2p session made, we are %s", net.B32Addr())

	log.Info("closing i2p session")
	net.Close()
	log.Info("done")
}
