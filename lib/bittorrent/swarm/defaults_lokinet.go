//go:build lokinet
// +build lokinet

package swarm

import "github.com/majestrate/XD/lib/bittorrent/extensions"

const DefaultMaxParallelRequests = 48
const DefaultPEXDialect = extensions.LokinetPeerExchange
