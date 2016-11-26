package metainfo

import (
	"os"
	"testing"
	"github.com/zeebo/bencode"
	"strings"
)

func TestLoadTorrent(t *testing.T) {
	f, err := os.Open("test.torrent")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	tf := new(TorrentFile)
	dec := bencode.NewDecoder(f)
	err = dec.Decode(tf)
	if err != nil {
		t.Error(err)
	}
	if strings.ToUpper(tf.Infohash().Hex()) != "E8E6FCDBD1E2B4DFE1D3192E50193FAA35AE44E3" {
		t.Error(tf.Infohash().Hex())
	}
	// TODO: check members
}

