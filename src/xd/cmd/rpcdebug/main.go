package main

import (
	"fmt"
	"net/rpc/jsonrpc"
	"os"
	"xd/lib/bittorrent/swarm"
	"xd/lib/config"
	"xd/lib/log"
)

func main() {
	fname := "torrents.ini"
	if len(os.Args) > 1 {
		fname = os.Args[1]
	}
	cfg := new(config.Config)
	err := cfg.Load(fname)
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	c, err := jsonrpc.Dial("tcp", cfg.RPC.Bind)

	if err != nil {
		log.Errorf("rpc error: %s", err)
		return
	}
	defer c.Close()

	var i int
	list := new(swarm.TorrentsList)
	err = c.Call(swarm.RPCListTorrents, &i, list)
	if err != nil {
		log.Errorf("rpc error: %s", err)
		return
	}

	var status swarm.TorrentStatus
	for _, ih := range list.Infohashes {
		var tx, rx float32
		err = c.Call(swarm.RPCTorrentStatus, &ih, &status)

		if err != nil {
			log.Errorf("rpc error: %s", err)
			return
		}
		fmt.Printf("swarm info for %s\n", ih)
		for _, peer := range status.Peers {
			fmt.Printf("%s tx=%f rx=%f\n", peer.ID.String(), peer.TX, peer.RX)
			tx += peer.TX
			rx += peer.RX
		}
		fmt.Printf("\ntotal tx=%f rx=%f\n", tx, rx)
		fmt.Println()
	}
}
