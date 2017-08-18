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

var formatRate = util.FormatRate

func Run() {
	var args []string
	cmd := "list"
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
	c := rpc.NewClient(u.String())

	switch strings.ToLower(cmd) {
	case "list":
		listTorrents(c)
	case "add":
		addTorrents(c, args...)
	case "set-piece-window":
		setPieceWindow(c, args[0])
	}
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
		fmt.Printf("fetch %s", urls[idx])
		c.AddTorrent(urls[idx])
	}
}

func listTorrents(c *rpc.Client) {
	var err error
	var list swarm.TorrentsList
	list, err = c.ListTorrents()
	if err != nil {
		log.Errorf("rpc error: %s", err)
		return
	}
	var globalTx, globalRx float64

	var torrents swarm.TorrentStatusList
	sort.Stable(&list.Infohashes)

	for _, ih := range list.Infohashes {
		var status swarm.TorrentStatus
		status, err = c.SwarmStatus(ih)

		if err != nil {
			log.Errorf("rpc error: %s", err)
			return
		}

		torrents = append(torrents, status)

	}
	sort.Stable(&torrents)
	for _, status := range torrents {
		var tx, rx float64
		fmt.Printf("%s [%s] %s\n", status.Name, status.Infohash, status.Bitfield.Percent())
		sort.Stable(&status.Peers)
		for _, peer := range status.Peers {
			fmt.Printf("%s\t\t\ttx=%s rx=%s\n", peer.ID, formatRate(peer.TX), formatRate(peer.RX))
			tx += peer.TX
			rx += peer.RX
		}
		fmt.Printf("%s tx=%s rx=%s\n", status.State, formatRate(tx), formatRate(rx))
		fmt.Println("files:")
		for idx, f := range status.Files {
			fmt.Printf("\t[%d] %s (%s)\n", idx, f.FileInfo.Path.FilePath(), f.Progress.Percent())
		}
		fmt.Println()
		globalRx += rx
		globalTx += tx
	}
	fmt.Println()
	fmt.Printf("%d torrents: tx=%s rx=%s\n", list.Infohashes.Len(), formatRate(globalTx), formatRate(globalRx))
}
