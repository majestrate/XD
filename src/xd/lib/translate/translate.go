package translate

import (
	"gopkg.in/leonelquinteros/gotext.v1"
	"os"
)

// default locale to use
const DefaultLocale = "en_US"
const Domain = "default"

func init() {
	lc := os.Getenv(env)
	if lc == "" {
		lc = DefaultLocale
	}
	gotext.Configure(Path, lc, Domain)
}

var TN = gotext.GetN
var T = gotext.Get

/** convert error to string */
func E(err error) (str string) {
	if err != nil {
		str = T(err.Error())
	}
	return
}
