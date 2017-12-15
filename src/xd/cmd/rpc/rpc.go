package rpc

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"xd/lib/bittorrent/swarm"
	"xd/lib/config"
	"xd/lib/log"
	"xd/lib/rpc"
	"xd/lib/util"
)

func formatRate(r float64) string {
	str := util.FormatRate(r)
	for len(str) < 12 {
		str += " "
	}
	return str
}

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
	u := url.URL{
		Scheme: "http",
		Host:   cfg.RPC.Bind,
		Path:   rpc.RPCPath,
	}
	swarms := cfg.Bittorrent.Swarms
	count := 0
	switch strings.ToLower(cmd) {
	case "list":
		for count < swarms {
			c := rpc.NewClient(u.String(), count)
			listTorrents(c)
			count++
		}
	case "add":
		for count < swarms {
			c := rpc.NewClient(u.String(), count)
			addTorrents(c, args...)
			count++
		}
	case "start":
		for count < swarms {
			c := rpc.NewClient(u.String(), count)
			startTorrents(c, args...)
			count++
		}
	case "stop":
		for count < swarms {
			c := rpc.NewClient(u.String(), count)
			stopTorrents(c, args...)
			count++
		}
	case "remove":
		for count < swarms {
			c := rpc.NewClient(u.String(), count)
			removeTorrents(c, args...)
			count++
		}
	case "delete":
		for count < swarms {
			c := rpc.NewClient(u.String(), count)
			deleteTorrents(c, args...)
			count++
		}
	case "set-piece-window":
		for count < swarms {
			c := rpc.NewClient(u.String(), count)
			setPieceWindow(c, args[0])
			count++
		}
	case "help":
		printHelp(os.Args[0])
	}
}

func printHelp(cmd string) {
	fmt.Printf("usage: %s [help|list|add http://somesite.i2p/some.torrent|set-piece-window n|remove infohash|delete infohash|stop infohash|start infohash]", cmd)
	fmt.Println()
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
		fmt.Printf("fetch %s ... ", urls[idx])
		err := c.AddTorrent(urls[idx])
		if err == nil {
			fmt.Println("OK")
		} else {
			fmt.Println(err.Error())
		}
	}
}

func startTorrents(c *rpc.Client, ih ...string) {
	for idx := range ih {
		fmt.Printf("start %s ... ", ih[idx])
		err := c.AddTorrent(ih[idx])
		if err == nil {
			fmt.Println("OK")
		} else {
			fmt.Println(err.Error())
		}
	}
}

func stopTorrents(c *rpc.Client, ih ...string) {
	for idx := range ih {
		fmt.Printf("stop %s ... ", ih[idx])
		err := c.StopTorrent(ih[idx])
		if err == nil {
			fmt.Println("OK")
		} else {
			fmt.Println(err.Error())
		}
	}
}

func removeTorrents(c *rpc.Client, ih ...string) {
	for idx := range ih {
		fmt.Printf("remove %s ... ", ih[idx])
		err := c.RemoveTorrent(ih[idx])
		if err == nil {
			fmt.Println("OK")
		} else {
			fmt.Println(err.Error())
		}
	}
}

func deleteTorrents(c *rpc.Client, ih ...string) {
	for idx := range ih {
		fmt.Printf("delete %s ... ", ih[idx])
		err := c.DeleteTorrent(ih[idx])
		if err == nil {
			fmt.Println("OK")
		} else {
			fmt.Println(err.Error())
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
	var globalTx, globalRx float64

	var torrents swarm.TorrentStatusList
	for _, status := range st {
		torrents = append(torrents, status)
	}
	sort.Stable(&torrents)
	for _, status := range torrents {
		var tx, rx float64
		fmt.Printf("%s [%s] %.2f\n", status.Name, status.Infohash, status.Progress)
		fmt.Println("peers:")
		sort.Stable(&status.Peers)
		for _, peer := range status.Peers {
			pad := peer.ID

			for len(pad) < 65 {
				pad += " "
			}
			fmt.Printf("\t%stx=%s rx=%s\n", pad, formatRate(peer.TX), formatRate(peer.RX))
			tx += peer.TX
			rx += peer.RX
		}
		fmt.Printf("%s tx=%s rx=%s (%s)\n", status.State, formatRate(tx), formatRate(rx), formatRate(status.Ratio()))
		fmt.Println("files:")
		for idx, f := range status.Files {
			fmt.Printf("\t[%d] %s (%.2f)\n", idx, f.FileInfo.Path.FilePath(), f.Progress)
		}
		fmt.Println()
		globalRx += rx
		globalTx += tx
	}
	fmt.Println()
	fmt.Printf("%d torrents: tx=%s rx=%s (%s)\n", torrents.Len(), formatRate(globalTx), formatRate(globalRx), formatRate(torrents.Ratio()))
	fmt.Println()
	fmt.Println()
}
