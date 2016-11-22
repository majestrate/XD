package bittorrent

import (
	"io"
)

// bittorrent wire message
type WireMessage struct {

}

// recv from reader
func (msg *WireMessage) Recv(r io.Reader) (err error) {
	return
}

// send via writer
func (msg *WireMessage) Send(w io.Writer) (err error) {
	return
}
