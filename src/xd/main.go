package xd

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
	"xd/lib/bittorrent/swarm"
	"xd/lib/config"
	"xd/lib/log"
	"xd/lib/util"
	"xd/lib/version"
)

type httpRPC struct {
	w http.ResponseWriter
	r *http.Request
}

// Run runs XD main function
func Run() {
	v := version.Version()
	done := make(chan error)
	conf := new(config.Config)
	fname := "torrents.ini"
	if len(os.Args) > 1 {
		fname = os.Args[1]
	}
	if fname == "-h" || fname == "--help" {
		fmt.Fprintf(os.Stdout, "usage: %s [config.ini]\n", os.Args[0])
		return
	}
	if os.Getenv("PPROF") == "1" {
		go func() {
			log.Warnf("pprof exited: %s", http.ListenAndServe("127.0.0.1:6060", nil))
		}()
	}
	log.Infof("starting %s", v)
	var err error
	if !util.CheckFile(fname) {
		conf.Load(fname)
		err = conf.Save(fname)
		if err != nil {
			log.Errorf("failed to save initial config: %s", err)
			return
		}
		log.Infof("auto-generated new config at %s", fname)
	}
	err = conf.Load(fname)
	if err != nil {
		log.Errorf("failed to config %s", err)
		return
	}
	log.Infof("loaded config %s", fname)
	log.SetLevel(conf.Log.Level)
	st := conf.Storage.CreateStorage()

	sw := swarm.NewSwarm(st)

	go func() {
		// run swarm
		done <- sw.Run()
	}()

	go func() {
		ts, e := st.OpenAllTorrents()
		if e != nil {
			log.Errorf("error opening all torrents: %s", e)
			done <- e
			return
		}
		for _, t := range ts {
			e = sw.AddTorrent(t, false)
			if e != nil {
				log.Errorf("error adding torrent: %s", e)
				done <- e
				return
			}
		}
	}()

	// torrent auto adder
	go func() {
		for {
			nt := st.PollNewTorrents()
			for _, t := range nt {
				name := t.MetaInfo().TorrentName()
				e := sw.AddTorrent(t, true)
				if e == nil {
					log.Infof("added %s", name)
				} else {
					log.Errorf("Failed to add %s: %s", name, e)
				}
			}
			time.Sleep(time.Second)
		}
	}()

	// start rpc server
	if conf.RPC.Enabled {
		log.Infof("RPC enabled")
	}

	net := conf.I2P.CreateSession()
	log.Info("opening i2p session")
	err = net.Open()
	if err != nil {
		log.Fatalf("failed to open i2p session: %s", err.Error())
	}
	log.Infof("i2p session made, we are %s", net.B32Addr())
	sw.SetNetwork(net)
	err = <-done
	close(done)
	if err != nil {
		log.Errorf("error: %s", err)
	}
	// close network because we are done
	log.Info("closing i2p network connection")
	net.Close()
	log.Info("exited")
}
