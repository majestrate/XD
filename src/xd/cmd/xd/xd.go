package xd

import (
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"time"
	"xd/lib/bittorrent/swarm"
	"xd/lib/config"
	"xd/lib/log"
	"xd/lib/network/i2p"
	"xd/lib/rpc"
	t "xd/lib/translate"
	"xd/lib/util"
	"xd/lib/version"
)

type httpRPC struct {
	w http.ResponseWriter
	r *http.Request
}

func printHelp(cmd string) {
	log.Infof("usage: %s [config.ini] | --genconf config.ini\n", cmd)
}

// Run runs XD main function
func Run() {

	running := true
	var closers []io.Closer
	v := version.Version()
	conf := new(config.Config)
	fname := "torrents.ini"
	if len(os.Args) > 1 {
		fname = os.Args[1]
	}
	if fname == "-h" || fname == "--help" {
		printHelp(os.Args[0])
		return
	}
	var err error
	if fname == "--genconf" {
		if len(os.Args) == 3 {
			conf.Load("")
			err = conf.Save(os.Args[2])
			if err != nil {
				log.Errorf("failed to save config: %s", err)
			}
		} else {
			printHelp(os.Args[0])
		}
		return
	}

	log.Info(t.T("starting %s", v))
	if !util.CheckFile(fname) {
		conf.Load(fname)
		err = conf.Save(fname)
		if err != nil {
			log.Errorf("failed to save initial config: %s", err)
			return
		}
		log.Info(t.T("auto-generated new config at %s", fname))
	}
	err = conf.Load(fname)
	if err != nil {
		log.Errorf("failed to config %s", err)
		return
	}
	log.Info(t.T("loaded config %s", fname))
	log.SetLevel(conf.Log.Level)

	if conf.Log.Pprof {
		go func() {
			pprofaddr := "127.0.0.1:6060"
			log.Infof("spawning pprof at %s", pprofaddr)
			log.Warnf("pprof exited: %s", http.ListenAndServe(pprofaddr, nil))
		}()
	}

	st := conf.Storage.CreateStorage()
	err = st.Init()
	if err != nil {
		log.Errorf("error initializing storage: %s", err)
		return
	}
	closers = append(closers, st)
	var swarms []*swarm.Swarm
	count := 0
	for count < conf.Bittorrent.Swarms {
		gnutella := conf.Gnutella.CreateSwarm()
		sw := conf.Bittorrent.CreateSwarm(st, gnutella)
		if gnutella != nil {
			closers = append(closers, gnutella)
		}
		swarms = append(swarms, sw)
		closers = append(closers, sw)
		count++
	}

	ts, err := st.OpenAllTorrents()
	if err != nil {
		log.Errorf("error opening all torrents: %s", err)
		return
	}
	for _, t := range ts {
		for _, sw := range swarms {
			err = sw.AddTorrent(t)
			if err != nil {
				log.Errorf("error adding torrent: %s", err)
			}
		}
	}

	// torrent auto adder
	go func() {
		for running {
			nt := st.PollNewTorrents()
			for _, t := range nt {
				e := t.VerifyAll(true)
				if e != nil {
					log.Errorf("failed to add %s: %s", t.Name(), e.Error())
					continue
				}
				for _, sw := range swarms {
					sw.AddTorrent(t)
				}
			}
			time.Sleep(time.Second)
		}
	}()

	// start rpc server
	if conf.RPC.Enabled {
		log.Infof("RPC enabled")
		srv := rpc.NewServer(swarms)

		var l net.Listener
		var e error
		var closeSock func()
		if strings.HasPrefix(conf.RPC.Bind, "unix:") {
			sock := conf.RPC.Bind[5:]
			closeSock = func() {
				os.Remove(sock)
			}
			l, e = net.Listen("unix", sock)
			if e == nil {
				e = os.Chmod(sock, 0640)
			}
		} else {
			l, e = net.Listen("tcp", conf.RPC.Bind)
			closeSock = func() {
			}
		}
		if e == nil {
			closers = append(closers, l)
			s := http.Server{
				Handler: srv,
			}
			go func() {
				log.Errorf("rpc died: %s", s.Serve(l))
				closeSock()
			}()
		} else {
			log.Errorf("failed to bind rpc: %s", e)
		}
	}

	runFunc := func(n i2p.Session, sw *swarm.Swarm) {
		for sw.Running() {
			log.Info("opening i2p session")
			err := n.Open()
			if err == nil {
				log.Infof("i2p session made, we are %s", n.B32Addr())
				err = sw.Run(n)
				if err != nil {
					log.Errorf("lost i2p session: %s", err)
				}
			} else {
				log.Errorf("failed to create i2p session: %s", err)
				time.Sleep(time.Second)
			}
		}
	}

	for idx := range swarms {
		net := conf.I2P.CreateSession()
		go runFunc(net, swarms[idx])
		closers = append(closers, net)
	}
	sigchnl := make(chan os.Signal)
	signal.Notify(sigchnl, os.Interrupt)
	for {
		sig := <-sigchnl
		if sig == os.Interrupt {
			running = false
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
