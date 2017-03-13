package storage

import (
	"os"
	"path/filepath"
	"testing"
	"xd/lib/common"
	"xd/lib/metainfo"
)

func TestFS(t *testing.T) {
	tf := new(metainfo.TorrentFile)
	p := filepath.Join("test", "test.rand.bin.torrent")
	f, err := os.Open(p)
	if err != nil {
		t.Errorf("failed to open test file: %s", err)
		t.Fail()
		return
	}
	err = tf.BDecode(f)
	if err != nil {
		t.Errorf("failed to decode test file: %s", err)
		t.Fail()
		return
	}

	seed := filepath.Join("test", "seed")
	leech := filepath.Join("test", "leech")

	stSeed := &FsStorage{
		DataDir: filepath.Join(seed, "download"),
		MetaDir: filepath.Join(seed, "meta"),
	}

	err = stSeed.Init()
	if err != nil {
		t.Errorf("failed to init seed storage: %s", err)
		t.Fail()
		return
	}

	stLeech := &FsStorage{
		DataDir: filepath.Join(leech, "download"),
		MetaDir: filepath.Join(leech, "meta"),
	}

	err = stLeech.Init()

	if err != nil {
		t.Errorf("failed to init leech storage: %s", err)
		t.Fail()
		return
	}

	var seedTorrent Torrent

	seedTorrent, err = stSeed.OpenTorrent(tf)

	if err != nil {
		t.Errorf("failed to open seed torrent: %s", err)
		t.Fail()
		return
	}

	err = seedTorrent.VerifyAll(false)

	if err != nil {
		t.Errorf("failed to verify seed data: %s", err)
		t.Fail()
		return
	}

	var leechTorrent Torrent

	leechTorrent, err = stLeech.OpenTorrent(tf)

	if err != nil {
		t.Errorf("failed to open seed torrent: %s", err)
		t.Fail()
		return
	}

	err = leechTorrent.Allocate()
	if err != nil {
		t.Errorf("failed to allocate leech torrent: %s", err)
		t.Fail()
		return
	}

	err = leechTorrent.VerifyAll(true)

	if err != nil {
		t.Errorf("failed to verify initial leech data: %s", err)
		t.Fail()
		return
	}

	var req common.PieceRequest

	pCount := tf.Info.NumPieces()
	t.Logf("we have %d pieces", pCount)
	req.Length = tf.Info.PieceLength
	var pc *common.PieceData
	for err == nil && pCount > req.Index {
		pc, err = seedTorrent.GetPiece(&req)
		if err == nil {
			t.Logf("put piece idx=%d begin=%d len=%d", pc.Index, pc.Begin, len(pc.Data))
			err = leechTorrent.PutPiece(pc)
			if err == nil {
				req.Index++
			} else {
				t.Errorf("leech torrent put piece failed: %s", err)
			}
		} else {
			t.Errorf("seed torrent getpiece failed: %s", err)
		}
	}

	if err == nil {
		err = leechTorrent.VerifyAll(false)
	}

	if err != nil {
		t.Fail()
	}
}
