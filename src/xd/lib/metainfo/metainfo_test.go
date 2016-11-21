package metainfo

import (
	"fmt"
	"os"
	"testing"
	"github.com/zeebo/bencode"
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
	fmt.Printf("%s", tf)
}

