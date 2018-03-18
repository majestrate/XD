package dht

import (
	"bytes"
	"github.com/zeebo/bencode"
	"xd/lib/bittorrent/extensions"
	"xd/lib/common"
)

type XDHT struct {
}

func (dht *XDHT) HandleError(code, msg interface{}) {

}

func (dht *XDHT) HandleMessage(msg extensions.Message, src common.PeerID) (err error) {
	r := bytes.NewReader(msg.Raw)
	var dhtmsg Message
	err = bencode.NewDecoder(r).Decode(&dhtmsg)
	if err == nil {
		if dhtmsg.IsError() {
			dht.HandleError(dhtmsg.Error[0], dhtmsg.Error[1])
		}
	}
	return
}
