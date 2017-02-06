package xd

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"time"
	"xd/lib/bittorrent/swarm"
	"xd/lib/config"
	"xd/lib/log"
	"xd/lib/util"
)

func Run() {
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
	util.EnsureFile(fname, 0)
	err := conf.Load(fname)
	if err != nil {
		log.Errorf("failed to config %s", err)
		return
	}
	log.Info("loaded config")

	st := conf.Storage.CreateStorage()

	sw := swarm.NewSwarm(st)
	go func() {
		ts, err := st.OpenAllTorrents()
		if err != nil {
			log.Errorf("error opening all torrents: %s", err)
			done <- err
			return
		}
		for _, t := range ts {
			err := t.VerifyAll(true)
			if err != nil {
				log.Errorf("failed to verify: %s", err)
				done <- err
				return
			}
			err = sw.AddTorrent(t)
			if err != nil {
				log.Errorf("error adding torrent: %s", err)
				done <- err
				return
			}
		}
		// run swarm
		done <- sw.Run()
	}()

	// torrent auto adder
	go func() {
		for {
			nt := st.PollNewTorrents()
			for _, t := range nt {
				name := t.MetaInfo().TorrentName()
				log.Debugf("adding torrent %s", name)
				err := sw.AddTorrent(t)
				if err == nil {
					log.Infof("added %s", name)
				} else {
					log.Errorf("Failed to add %s: %s", name, err)
				}
			}
			time.Sleep(time.Second)
		}
	}()

	// start rpc
	if conf.RPC.Enabled {
		log.Infof("RPC enabled")
		go func() {
			r := new(rpc.Server)
			er := r.Register(sw.GetRPC())
			if er != nil {
				log.Errorf("rpc register error: %s", er)
				return
			}
			r.HandleHTTP("/", "/debug")
			l, e := net.Listen("tcp", conf.RPC.Bind)
			if e == nil {
				var c net.Conn
				for e == nil {
					c, e = l.Accept()
					go r.ServeCodec(jsonrpc.NewServerCodec(c))
				}
			} else {
				log.Warnf("failed to start rpc: %s", e)
			}
		}()
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
