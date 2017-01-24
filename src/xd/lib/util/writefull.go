package util

import "io"

// ensure a byteslice is written in full
func WriteFull(w io.Writer, d []byte) (err error) {
	var n int
	for n < len(d) {
		var o int
		o, err = w.Write(d[n:])
		if err == nil {
			n += o
		} else {
			break
		}
	}
	return
}
