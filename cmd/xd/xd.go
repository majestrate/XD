package xd

import (
	"bufio"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"time"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
	"github.com/majestrate/XD/lib/config"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/rpc"
	"github.com/majestrate/XD/lib/sync"
	t "github.com/majestrate/XD/lib/translate"
	"github.com/majestrate/XD/lib/util"
	"github.com/majestrate/XD/lib/version"
)

type httpRPC struct {
	w http.ResponseWriter
	r *http.Request
}

func printHelp(cmd string) {
	log.Infof("usage: %s [config.ini] | --genconf config.ini\n", cmd)
}

func NewContext() *Context {
	pr, pw := io.Pipe()
	return &Context{
		pr:      pr,
		pw:      pw,
		sigchnl: make(chan os.Signal),
		netlost: true,
	}
}

type Context struct {
	pr         io.ReadCloser
	pw         io.WriteCloser
	closers    sync.Map
	numClosers int
	quit       bool
	swarms     []*swarm.Swarm
	sigchnl    chan os.Signal
	netlost    bool
}

func (c *Context) Run() {
	go func() {
		io.Copy(c.pw, os.Stdin)
	}()
	r := bufio.NewReader(c.pr)
	for {
		line, err := r.ReadString(10)
		if err == nil {
			if strings.ToLower(line) == "f\n" {
				if c.netlost {
					log.Debug("respec paid")
				}
			} else if line == "\n" && c.quit {
				break
			} else if line == "" {
				break
			}
		} else {
			break
		}
	}
	c.pr.Close()
}

func (c *Context) Running() bool {
	return !c.quit
}

func (c *Context) RunSignals() {
	signal.Notify(c.sigchnl, os.Interrupt)
	for {
		sig := <-c.sigchnl
		if sig == os.Interrupt {
			log.Info("Interrupted")
			c.Close()
			return
		} else {
			log.Warnf("got wierd signal wtf: %s", sig)
			continue
		}
	}
}

func (c *Context) AddCloser(cl io.Closer) int {
	c.numClosers++
	c.closers.Store(c.numClosers, cl)
	return c.numClosers
}

func (c *Context) RemoveCloser(id int) {
	c.closers.Delete(id)
}

func (c *Context) ReplaceCloser(id int, cl io.Closer) {
	c.closers.Store(id, cl)
}

func (c *Context) AddSwarm(sw *swarm.Swarm) {
	c.swarms = append(c.swarms, sw)
}

func (c *Context) Close() error {
	c.quit = true
	c.pw.Close()
	// close swarms first
	for _, sw := range c.swarms {
		sw.Close()
	}
	c.closers.Range(func(k, v interface{}) bool {
		cl := v.(io.Closer)
		cl.Close()
		return true
	})
	return nil
}

// Run runs XD main function
func Run() {

	ctx := NewContext()

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
			l, err := net.Listen("tcp", pprofaddr)
			if err == nil {
				ctx.AddCloser(l)
				log.Infof("spawning pprof at %s", pprofaddr)
				log.Warnf("pprof exited: %s", http.Serve(l, nil))
			}
		}()
	}

	st := conf.Storage.CreateStorage()
	err = st.Init()
	if err != nil {
		log.Errorf("error initializing storage: %s", err)
		return
	}
	// start io thread
	go st.Run()
	count := 0
	for count < conf.Bittorrent.Swarms {
		gnutella := conf.Gnutella.CreateSwarm()
		sw := conf.Bittorrent.CreateSwarm(st, gnutella)
		if gnutella != nil {
			ctx.AddCloser(gnutella)
		}
		ctx.AddSwarm(sw)
		count++
	}

	ts, err := st.OpenAllTorrents()
	if err != nil {
		log.Errorf("error opening all torrents: %s", err)
		return
	}
	for _, t := range ts {
		for _, sw := range ctx.swarms {
			err = sw.AddTorrent(t)
			if err != nil {
				log.Errorf("error adding torrent: %s", err)
			}
		}
	}

	// torrent auto adder
	go func() {
		for ctx.Running() {
			nt := st.PollNewTorrents()
			for _, t := range nt {
				e := t.VerifyAll()
				if e != nil {
					log.Errorf("failed to add %s: %s", t.Name(), e.Error())
					continue
				}
				for _, sw := range ctx.swarms {
					sw.AddTorrent(t)
				}
			}
			time.Sleep(time.Second)
		}
	}()

	// start rpc server
	if conf.RPC.Enabled {
		log.Infof("RPC enabled")
		var host string
		var l net.Listener
		var e error
		var cleanSock func()
		if strings.HasPrefix(conf.RPC.Bind, "unix:") {
			sock := conf.RPC.Bind[5:]
			cleanSock = func() {
				os.Remove(sock)
			}
			l, e = net.Listen("unix", sock)
			if e == nil {
				e = os.Chmod(sock, 0640)
			}
		} else {
			l, e = net.Listen("tcp", conf.RPC.Bind)
			cleanSock = func() {
			}
			host = conf.RPC.ExpectedHost
		}
		if e == nil {
			ctx.AddCloser(l)
			s := &http.Server{
				Handler: rpc.NewServer(ctx.swarms, host),
			}
			go func(serv *http.Server) {
				log.Errorf("rpc died: %s", serv.Serve(l))
				cleanSock()
			}(s)
		} else {
			log.Errorf("failed to bind rpc: %s", e)
		}
	}

	runLokiNetFunc := func(netConf config.LokiNetConfig, sw *swarm.Swarm) {
		for sw.Running() {
			n, err := netConf.CreateSession()
			if err != nil {
				log.Infof("failed to create lokinet session: %s", err.Error())
				time.Sleep(time.Second)
				continue
			}
			id := ctx.AddCloser(n)
			log.Info("opening lokinet session")
			err = n.Open()
			if err == nil {
				log.Infof("we up at %s", n.LocalName())
				sw.ObtainedNetwork(n)
				ctx.netlost = false
				err = sw.Run()
				if err != nil {
					ctx.netlost = true
					log.Errorf("lost lokinet session: %s", err)
					sw.LostNetwork()
					ctx.RemoveCloser(id)
				}
			} else {
				ctx.netlost = true
				ctx.RemoveCloser(id)
				log.Errorf("failed to open lokinet session: %s", err)
				time.Sleep(time.Second)
			}
		}
	}

	runI2PFunc := func(netConf config.I2PConfig, sw *swarm.Swarm) {
		n := netConf.CreateSession()
		id := ctx.AddCloser(n)
		for sw.Running() {
			log.Info("opening i2p session")
			err := n.Open()
			if err == nil {
				log.Infof("i2p session made, we are %s", n.B32Addr())
				sw.ObtainedNetwork(n)
				ctx.netlost = false
				err = sw.Run()
				if err != nil {
					ctx.netlost = true
					log.Errorf("lost i2p session: %s", err)
					sw.LostNetwork()
					n = netConf.CreateSession()
					ctx.ReplaceCloser(id, n)
				}
			} else {
				ctx.netlost = true
				n = netConf.CreateSession()
				ctx.ReplaceCloser(id, n)
				log.Errorf("failed to create i2p session: %s", err)
				time.Sleep(time.Second)
			}
		}
	}

	for idx := range ctx.swarms {
		if conf.I2P.Disabled {
			if !conf.LokiNet.Disabled {
				go runLokiNetFunc(conf.LokiNet, ctx.swarms[idx])
			}
		} else {
			go runI2PFunc(conf.I2P, ctx.swarms[idx])
		}
	}
	ctx.AddCloser(st)
	go ctx.RunSignals()
	ctx.Run()
}
