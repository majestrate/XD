package rpc

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
	"github.com/majestrate/XD/lib/config"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/rpc"
	t "github.com/majestrate/XD/lib/translate"
	"github.com/majestrate/XD/lib/util"
	"github.com/majestrate/XD/lib/version"
)

func formatRate(r float64) string {
	str := util.FormatRate(r)
	for len(str) < 12 {
		str += " "
	}
	return str
}

// Run runs xd-cli main function
func Run() {
	var args []string
	cmd := "help"
	fname := "torrents.ini"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
		args = os.Args[2:]
	}
	cfg := new(config.Config)
	err := cfg.Load(fname)
	if err != nil {
		log.Errorf("error: %s", err)
		return
	}
	log.SetLevel(cfg.Log.Level)
	var rpcURL string
	if strings.HasPrefix(cfg.RPC.Bind, "unix:") {
		rpcURL = cfg.RPC.Bind
	} else {
		u := url.URL{
			Scheme: "http",
			Host:   cfg.RPC.Bind,
			Path:   rpc.RPCPath,
		}
		rpcURL = u.String()
	}
	swarms := cfg.Bittorrent.Swarms
	count := 0
	switch strings.ToLower(cmd) {
	case "list":
		for count < swarms {
			c := rpc.NewClient(rpcURL, count)
			listTorrents(c)
			count++
		}
	case "add":
		for count < swarms {
			c := rpc.NewClient(rpcURL, count)
			addTorrents(c, args...)
			count++
		}
	case "start":
		for count < swarms {
			c := rpc.NewClient(rpcURL, count)
			startTorrents(c, args...)
			count++
		}
	case "stop":
		for count < swarms {
			c := rpc.NewClient(rpcURL, count)
			stopTorrents(c, args...)
			count++
		}
	case "remove":
		for count < swarms {
			c := rpc.NewClient(rpcURL, count)
			removeTorrents(c, args...)
			count++
		}
	case "delete":
		for count < swarms {
			c := rpc.NewClient(rpcURL, count)
			deleteTorrents(c, args...)
			count++
		}
	case "set-piece-window":
		for count < swarms {
			c := rpc.NewClient(rpcURL, count)
			setPieceWindow(c, args[0])
			count++
		}
	case "version":
		fmt.Println(version.Version())
	case "help":
		printHelp(os.Args[0])
	}
}

func printHelp(cmd string) {
	fmt.Println(t.T("usage: %s [help|version|list|add http://somesite.i2p/some.torrent|set-piece-window n|remove infohash|delete infohash|stop infohash|start infohash]", cmd))
}

func setPieceWindow(c *rpc.Client, str string) {
	n, err := strconv.Atoi(str)
	if err != nil {
		log.Fatalf("error: %s", err.Error())
	}
	c.SetPieceWindow(n)
}

func addTorrents(c *rpc.Client, urls ...string) {
	for idx := range urls {
		fmt.Println(t.T("fetch %s ... ", urls[idx]))
		err := c.AddTorrent(urls[idx])
		if err == nil {
			fmt.Println(t.T("OK"))
		} else {
			fmt.Println(t.E(err))
		}
	}
}

func startTorrents(c *rpc.Client, ih ...string) {
	for idx := range ih {
		fmt.Println(t.T("start %s ... ", ih[idx]))
		err := c.AddTorrent(ih[idx])
		if err == nil {
			fmt.Println(t.T("OK"))
		} else {
			fmt.Println(t.E(err))
		}
	}
}

func stopTorrents(c *rpc.Client, ih ...string) {
	for idx := range ih {
		fmt.Println(t.T("stop %s ... ", ih[idx]))
		err := c.StopTorrent(ih[idx])
		if err == nil {
			fmt.Println(t.T("OK"))
		} else {
			fmt.Println(t.E(err))
		}
	}
}

func removeTorrents(c *rpc.Client, ih ...string) {
	for idx := range ih {
		fmt.Println(t.T("remove %s ... ", ih[idx]))
		err := c.RemoveTorrent(ih[idx])
		if err == nil {
			fmt.Println(t.T("OK"))
		} else {
			fmt.Println(t.E(err))
		}
	}
}

func deleteTorrents(c *rpc.Client, ih ...string) {
	for idx := range ih {
		fmt.Println(t.T("delete %s ... ", ih[idx]))
		err := c.DeleteTorrent(ih[idx])
		if err == nil {
			fmt.Println(t.T("OK"))
		} else {
			fmt.Println(t.E(err))
		}
	}
}

func listTorrents(c *rpc.Client) {
	var err error
	var st swarm.SwarmStatus
	st, err = c.GetSwarmStatus()
	if err != nil {
		log.Errorf("rpc error: %s", err)
		return
	}

	var torrents swarm.TorrentStatusList
	for _, status := range st {
		torrents = append(torrents, status)
	}
	sort.Stable(&torrents)
	for _, status := range torrents {
		fmt.Printf("%s [%s] %s %.2f\n", status.Name, status.Infohash, t.T("progress:"), status.Progress*100)
		fmt.Println(t.T("peers:"))
		sort.Stable(&status.Peers)
		for _, peer := range status.Peers {
			pad := peer.ID

			for len(pad) < 65 {
				pad += " "
			}
			fmt.Printf("\t%stx=%s rx=%s\n", pad, formatRate(peer.TX), formatRate(peer.RX))
		}
		fmt.Printf("%s tx=%s rx=%s (%s: %.2f)\n", status.State, formatRate(status.Peers.TX()), formatRate(status.Peers.RX()), t.T("ratio"), status.Ratio())
		fmt.Println(t.T("files:"))
		for idx, f := range status.Files {
			fmt.Printf("\t[%d] %s (%s: %.2f)\n", idx, f.FileInfo.Path.FilePath(""), t.T("progress:"), f.Progress)
		}
		fmt.Println()
	}
	fmt.Println()
	tx, rx := st.TotalSpeed()
	fmt.Printf("%s: tx=%s rx=%s (%.2f ratio)\n", t.TN("%d torrent", "%d torrents", torrents.Len(), torrents.Len()), formatRate(tx), formatRate(rx), st.Ratio())
	fmt.Println()
	fmt.Println()
}
