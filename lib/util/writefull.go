package util

import (
	"github.com/majestrate/XD/lib/log"
	"io"
)

// ensure a byteslices is written in full
func WriteFull(w io.Writer, d []byte) (err error) {
	var n int
	l := len(d)
	for n < l {
		var o int
		o, err = w.Write(d[n:])
		if err == nil {
			log.Debugf("wrote %d of %d", o, l)
			n += o
		} else {
			break
		}
	}
	return
}
