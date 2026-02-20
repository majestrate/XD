//go:build !lokinet
// +build !lokinet

package config

const DisableLokinetByDefault = true
const DisableI2PByDefault = false

// TODO: idk if these are the right names but the URL are correct
var DefaultOpenTrackers = map[string]string{
	"dg-opentracker":       "http://w7tpbzncbcocrqtwwm3nezhnnsw4ozadvi2hmvzdhrqzfxfum7wa.b32.i2p/a",
	// "chudo-opentracker":    "http://swhb5i7wcjcohmus3gbt3w6du6pmvl3isdvxvepuhdxxkfbzao6q.b32.i2p/a",  dead 2022-04-05
	"r4sas-opentracker":    "http://punzipidirfqspstvzpj6gb4tkuykqp6quurj6e23bgxcxhdoe7q.b32.i2p/a",
	// "thebland-opentracker": "http://s5ikrdyjwbcgxmqetxb3nyheizftms7euacuub2hic7defkh3xhq.b32.i2p/a",  dead 2020-09-13
	"skank-opentracker":    "http://by7luzwhx733fhc5ug2o75dcaunblq2ztlshzd7qvptaoa73nqua.b32.i2p/a",
	"postman-opentracker":    "http://6a4kxkg5wp33p25qqhgwl6sj4yh4xuf5b3p3qldwgclebchm3eea.b32.i2p/announce.php",
	"fattydove-opentracker":    "http://svece3bxv4vqlt2zuut5ww4ztkwunfcnab55pmnjjb6zfei3noha.b32.i2p/a",
	"sigmatracker-opentracker":    "http://qimlze77z7w32lx2ntnwkuqslrzlsqy7774v3urueuarafyqik5a.b32.i2p/a",
}
