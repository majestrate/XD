package xd

import (
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"
	"xd/lib/config"
	"xd/lib/log"
	"xd/lib/rpc"
	"xd/lib/util"
	"xd/lib/version"
)

type httpRPC struct {
	w http.ResponseWriter
	r *http.Request
}

// Run runs XD main function
func Run() {

	var closers []io.Closer
	v := version.Version()
	conf := new(config.Config)
	fname := "torrents.ini"
	if len(os.Args) > 1 {
		fname = os.Args[1]
	}
	if fname == "-h" || fname == "--help" {
		fmt.Fprintf(os.Stdout, "usage: %s [config.ini]\n", os.Args[0])
		return
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

	if conf.Log.Pprof {
		go func() {
			pprofaddr := "127.0.0.1:6060"
			log.Infof("spawning pprof at %s", pprofaddr)
			log.Warnf("pprof exited: %s", http.ListenAndServe(pprofaddr, nil))
		}()
	}

	st := conf.Storage.CreateStorage()
	sw := conf.Bittorrent.CreateSwarm(st)
	closers = append(closers, sw, st)

	ts, err := st.OpenAllTorrents()
	if err != nil {
		log.Errorf("error opening all torrents: %s", err)
		return
	}
	for _, t := range ts {
		err = sw.AddTorrent(t, false)
		if err != nil {
			log.Errorf("error adding torrent: %s", err)
			return
		}
	}

	// torrent auto adder
	go func() {
		for sw.Running() {
			nt := st.PollNewTorrents()
			for _, t := range nt {
				sw.AddTorrent(t, true)
			}
			time.Sleep(time.Second)
		}
	}()

	// start rpc server
	if conf.RPC.Enabled {
		log.Infof("RPC enabled")
		srv := rpc.NewServer(sw)
		go func() {
			log.Errorf("rpc died: %s", http.ListenAndServe(conf.RPC.Bind, srv))
		}()

	}

	net := conf.I2P.CreateSession()
	// network mainloop
	go func() {
		for sw.Running() {
			log.Info("opening i2p session")
			err := net.Open()
			if err == nil {
				log.Infof("i2p session made, we are %s", net.B32Addr())
				err = sw.Run(net)
				if err != nil {
					log.Errorf("lost i2p session: %s", err)
				}
			} else {
				log.Errorf("failed to create i2p session: %s", err)
				time.Sleep(time.Second)
			}
		}
	}()
	closers = append(closers, net)
	sigchnl := make(chan os.Signal)
	signal.Notify(sigchnl, os.Interrupt)
	for {
		sig := <-sigchnl
		if sig == os.Interrupt {
			log.Info("Interrupted")
			for idx := range closers {
				closers[idx].Close()
			}
			return
		} else {
			log.Warnf("got wierd signal wtf: %s", sig)
			continue
		}
	}
}
