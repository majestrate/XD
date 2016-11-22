package storage

import (
	"os"
	"path/filepath"
	"xd/lib/common"
	"xd/lib/metainfo"
)

// filesystem based storrent storage session
type fsTorrent struct {
	meta *metainfo.TorrentFile
}


func (t *fsTorrent) Allocate() (err error) {
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


func (st *FsStorage) OpenTorrent(info *metainfo.TorrentFile) (t Torrent, err error) {
	basepath := filepath.Join(st.DataDir, info.Info.Path)
	if ! info.IsSingleFile() {
		// create directory
		err = os.Mkdir(basepath, 0700)
	}
	if err == nil {

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
				meta: info,
			}
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
		f, err = os.Open(filepath.Join(st.MetaDir, m))
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
