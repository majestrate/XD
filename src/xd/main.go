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

// Run runs XD main function
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
	var err error
	if !util.CheckFile(fname) {
		err = conf.Save(fname)
		if err != nil {
			log.Errorf("failed to save initial config: %s", err)
			return
		}
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
		ts, e := st.OpenAllTorrents()
		if e != nil {
			log.Errorf("error opening all torrents: %s", e)
			done <- e
			return
		}
		for _, t := range ts {
			e = t.VerifyAll(false)
			if e != nil {
				log.Errorf("failed to verify: %s", e)
				done <- e
				return
			}
			e = sw.AddTorrent(t)
			if e != nil {
				log.Errorf("error adding torrent: %s", e)
				done <- e
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
				e := t.VerifyAll(true)
				if e != nil {
					log.Warnf("Failed to verify %s, %s", name, e)
				}
				e = sw.AddTorrent(t)
				if e == nil {
					log.Infof("added %s", name)
				} else {
					log.Errorf("Failed to add %s: %s", name, e)
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
			er := r.RegisterName(swarm.RPCName, sw.GetRPC())
			if er != nil {
				log.Errorf("rpc register error: %s", er)
				return
			}
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
