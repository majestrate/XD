package storage

import (
	"os"
	"path/filepath"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/metainfo"
	"xd/lib/util"
)

// filesystem based storrent storage session
type fsTorrent struct {
	// parent storage
	st *FsStorage
	// infohash
	ih common.Infohash
	// metainfo
	meta *metainfo.TorrentFile
}


func (t *fsTorrent) Allocate() (err error) {
	log.Infof("allocate files for %s", t.meta.TorrentName())
	return
}

func (t *fsTorrent) Bitfield() (bf *bittorrent.Bitfield) {
	if ! t.st.HasBitfield(t.ih) {
		// we have no pieces
		t.st.CreateNewBitfield(t.ih, len(t.meta.Info.Pieces))
	}
	return t.st.FindBitfield(t.ih)
}

func (t *fsTorrent) DownloadRemaining() (r int64) {
	bf := t.Bitfield()
	have := bf.CountSet() * t.meta.Info.PieceLength
	r = t.meta.Info.TotalSize() - have
	return
}

func (t *fsTorrent) MetaInfo() *metainfo.TorrentFile {
	return t.meta
}

func (t *fsTorrent) Infohash() (ih common.Infohash) {
	copy(ih[:], t.ih[:])
	return
}

func (t *fsTorrent) GetPiece(num, off int64) (p *common.Piece) {
	return 
}

func (t *fsTorrent) PutPiece(p *common.Piece) {

}

func (t *fsTorrent) Verify(piece int64) (err error) {
	return
}

func (t *fsTorrent) VerifyAll() (err error) {
	log.Infof("verify all pieces for %s", t.meta.TorrentName())
	pieces := len(t.meta.Info.Pieces)
	for pieces > 0 {
		pieces --
		err = t.Verify(int64(pieces))
		if err != nil {
			break
		}
	}
	return
}


// filesystem based torrent storage
type FsStorage struct {
	// directory for downloaded data
	DataDir string
	// directory for torrent seed data
	MetaDir string
}

func (st *FsStorage) Init() (err error) {
	log.Info("Ensure filesystem storage")
	err = util.EnsureDir(st.DataDir)
	if err == nil {
		err = util.EnsureDir(st.MetaDir)
	}
	return
}

func (st *FsStorage) FindBitfield(ih common.Infohash) (bf *bittorrent.Bitfield) {
	fpath := st.bitfieldFilename(ih)
	f, err := os.Open(fpath)
	if err == nil {
		bf = new(bittorrent.Bitfield)
		err = bf.BDecode(f)
		if err != nil {
			bf = nil
		}
		f.Close()
	}
	return
}

func (st *FsStorage) bitfieldFilename(ih common.Infohash) string {
	return filepath.Join(st.MetaDir, ih.Hex()+".bitfield")
}

func (st *FsStorage) HasBitfield(ih common.Infohash) bool {
	_, err := os.Stat(st.bitfieldFilename(ih))
	return err == nil
}

func (st *FsStorage) CreateNewBitfield(ih common.Infohash, bits int) {
	fname := st.bitfieldFilename(ih)
	bf := bittorrent.NewBitfield(bits, nil)
	f, err := os.OpenFile(fname, os.O_CREATE | os.O_WRONLY, 0600)
	if err == nil {
		bf.BEncode(f)
		f.Close()
	}
}

func (st *FsStorage) OpenTorrent(info *metainfo.TorrentFile) (t Torrent, err error) {
	basepath := filepath.Join(st.DataDir, info.TorrentName())
	if ! info.IsSingleFile() {
		// create directory
		os.Mkdir(basepath, 0700)
	}
	
	ih := info.Infohash()
	metapath := filepath.Join(st.MetaDir, ih.Hex() + ".torrent")
	_, err = os.Stat(metapath)
	
	if os.IsNotExist(err) {
		// put meta info down onto filesystem
		var f *os.File
		f, err = os.OpenFile(metapath, os.O_CREATE | os.O_WRONLY, 0600)
			if err == nil {
				err = info.BEncode(f)
				f.Close()
			}
	}
	
	if err == nil {
		t = &fsTorrent{
			st: st,
			meta: info,
			ih: ih,
		}
	}
	
	return
}

func (st *FsStorage) OpenAllTorrents() (torrents []Torrent, err error) {
	var matches []string
	matches, err = filepath.Glob(filepath.Join(st.MetaDir, "*.torrent"))
	for _, m := range matches {
		var t Torrent
		var f *os.File
		tf := new(metainfo.TorrentFile)
		f, err = os.Open(m)
		if err == nil {
			err = tf.BDecode(f)
			f.Close()
		}
		if err == nil {
			t, err = st.OpenTorrent(tf)
		}
		if t != nil {
			torrents = append(torrents, t)
		}
	}
	return
}

func (st *FsStorage) PollNewTorrents() (torrents []Torrent) {
	matches, _ := filepath.Glob(filepath.Join(st.DataDir, "*.torrent"))
	for _, m := range matches {
		var t Torrent
		tf := new(metainfo.TorrentFile)
		f, err := os.Open(m)
		if err == nil {
			err = tf.BDecode(f)
			f.Close()
		}
		if err != nil {
			log.Warnf("error checking torrent file: %s", err)
		}
		if st.HasBitfield(tf.Infohash()) {
			// we already have this torrent
			continue
		}
		if err == nil {
			t, err = st.OpenTorrent(tf)
		}
		if t != nil {
			torrents = append(torrents, t)
		}
	}
	return
}
