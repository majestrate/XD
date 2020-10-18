package dht

import (
	"bytes"
	"github.com/zeebo/bencode"
	"github.com/majestrate/XD/lib/bittorrent/extensions"
	"github.com/majestrate/XD/lib/common"
)

type XDHT struct {
}

func (dht *XDHT) HandleError(err *Error) {

}

func (dht *XDHT) HandleMessage(msg extensions.Message, src common.PeerID) (err error) {
	r := bytes.NewReader(msg.PayloadRaw)
	var dhtmsg Message
	err = bencode.NewDecoder(r).Decode(&dhtmsg)
	if err == nil {
		if dhtmsg.IsError() {
			dht.HandleError(dhtmsg.Err)
		}
	}
	return
}
