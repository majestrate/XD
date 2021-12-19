// +build !lokinet

package config

const DisableLokinetByDefault = true
const DisableI2PByDefault = false

// TODO: idk if these are the right names but the URL are correct
var DefaultOpenTrackers = map[string]string{
	"dg-opentracker":       "http://w7tpbzncbcocrqtwwm3nezhnnsw4ozadvi2hmvzdhrqzfxfum7wa.b32.i2p/a",
	"thebland-opentracker": "http://s5ikrdyjwbcgxmqetxb3nyheizftms7euacuub2hic7defkh3xhq.b32.i2p/a",
	"skank-opentracker":    "http://by7luzwhx733fhc5ug2o75dcaunblq2ztlshzd7qvptaoa73nqua.b32.i2p/a",
	"chudo-opentracker":    "http://by7luzwhx733fhc5ug2o75dcaunblq2ztlshzd7qvptaoa73nqua.b32.i2p/a",
}
