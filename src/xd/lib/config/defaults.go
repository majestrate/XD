// +build !lokinet

package config

const DisableLokinetByDefault = true
const DisableI2PByDefault = false

// TODO: idk if these are the right names but the URL are correct
var DefaultOpenTrackers = map[string]string{
	"dg-opentracker":       "http://w7tpbzncbcocrqtwwm3nezhnnsw4ozadvi2hmvzdhrqzfxfum7wa.b32.i2p/a",
	"thebland-opentracker": "http://s5ikrdyjwbcgxmqetxb3nyheizftms7euacuub2hic7defkh3xhq.b32.i2p/a",
	"psi-chihaya":          "http://uajd4nctepxpac4c4bdyrdw7qvja2a5u3x25otfhkptcjgd53ioq.b32.i2p/announce",
}
