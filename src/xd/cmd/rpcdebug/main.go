package main

import (
	"fmt"
	"net/rpc/jsonrpc"
	"os"
	"xd/lib/bittorrent/swarm"
	"xd/lib/config"
	"xd/lib/log"
)

var formatUnits = map[int]string{
	0: "B",
	1: "KB",
	2: "MB",
	3: "GB",
}

func formatRate(rate float32) string {
	r := uint32(rate)
	idx := 0
	for r > 1024 {
		r /= 1024
		idx++
	}
	return fmt.Sprintf("%d %s/s", r, formatUnits[idx])
}

func main() {
	fname := "torrents.ini"
	if len(os.Args) > 1 {
		fname = os.Args[1]
	}
	cfg := new(config.Config)
	err := cfg.Load(fname)
	if err != nil {
		log.Errorf("error: %s", err)
		return
	}
	log.SetLevel(cfg.Log.Level)
	c, err := jsonrpc.Dial("tcp", cfg.RPC.Bind)

	if err != nil {
		log.Errorf("rpc error: %s", err)
		return
	}
	defer c.Close()

	var i int
	var list swarm.TorrentsList
	log.Debugf("call %s", swarm.RPCListTorrents)
	err = c.Call(swarm.RPCListTorrents, &i, &list)
	if err != nil {
		log.Errorf("rpc error: %s", err)
		return
	}
	var globalTx, globalRx float32
	var status swarm.TorrentStatus
	for _, ih := range list.Infohashes {
		var tx, rx float32
		log.Debugf("call %s", swarm.RPCTorrentStatus)
		err = c.Call(swarm.RPCTorrentStatus, &ih, &status)

		if err != nil {
			log.Errorf("rpc error: %s", err)
			return
		}
		fmt.Printf("swarm info for %s\n", ih)
		for _, peer := range status.Peers {
			fmt.Printf("%s tx=%s rx=%s\n", peer.ID.String(), formatRate(peer.TX), formatRate(peer.RX))
			tx += peer.TX
			rx += peer.RX
		}
		fmt.Printf("\nswarm tx=%s rx=%s\n", formatRate(tx), formatRate(rx))
		fmt.Println()
		globalRx += rx
		globalTx += tx
	}
	fmt.Println()
	fmt.Printf("Total: tx=%s rx=%s\n", formatRate(globalTx), formatRate(globalRx))
}
