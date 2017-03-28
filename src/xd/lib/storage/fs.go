package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
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
	// cached bitfield
	bf *bittorrent.Bitfield
	// mutex for bitfield access
	bfmtx sync.RWMutex
}

func (t *fsTorrent) AllocateFile(f metainfo.FileInfo) (err error) {
	fname := filepath.Join(t.FilePath(), f.Path.FilePath())
	err = util.EnsureFile(fname, f.Length)
	return
}

func (t *fsTorrent) Allocate() (err error) {
	log.Infof("allocate files for %s", t.meta.TorrentName())
	if t.meta.IsSingleFile() {
		log.Debugf("file is %d bytes", t.meta.Info.Length)
		err = util.EnsureFile(t.FilePath(), t.meta.Info.Length)
	} else {
		for _, f := range t.meta.Info.Files {
			err = t.AllocateFile(f)
			if err != nil {
				break
			}
		}
	}
	return
}

func (t *fsTorrent) Bitfield() *bittorrent.Bitfield {
	t.bfmtx.Lock()
	if t.bf == nil {
		if !t.st.HasBitfield(t.ih) {
			// we have no pieces
			t.st.CreateNewBitfield(t.ih, t.meta.Info.NumPieces())
		}
		t.bf = t.st.FindBitfield(t.ih)
	}
	t.bfmtx.Unlock()
	return t.bf
}

func (t *fsTorrent) DownloadRemaining() (r uint64) {
	bf := t.Bitfield()
	have := uint64(bf.CountSet()) * uint64(t.meta.Info.PieceLength)
	r = t.meta.TotalSize() - have
	return
}

func (t *fsTorrent) MetaInfo() *metainfo.TorrentFile {
	return t.meta
}

func (t *fsTorrent) Name() string {
	return t.meta.TorrentName()
}

func (t *fsTorrent) Infohash() (ih common.Infohash) {
	copy(ih[:], t.ih[:])
	return
}

func (t *fsTorrent) FilePath() string {
	return filepath.Join(t.st.DataDir, t.meta.Info.Path)
}

func (t *fsTorrent) GetPiece(r *common.PieceRequest) (p *common.PieceData, err error) {

	files := t.meta.Info.GetFiles()
	sz := t.meta.Info.PieceLength

	pc := &common.PieceData{
		Index: r.Index,
		Begin: r.Begin,
		Data:  make([]byte, r.Length),
	}
	offset := (uint64(r.Index) * uint64(sz)) + uint64(r.Begin)
	pos := uint64(0)
	var at int64
	if !t.meta.IsSingleFile() {
		at = -1
	}
	readbuf := pc.Data[:]
	log.Debugf("offset=%d idx=%d begin=%d", offset, pc.Index, pc.Begin)
	for _, file := range files {
		log.Debugf("file.Length=%d", file.Length)
		fp := file.Path.FilePath()
		if pos+file.Length < offset && at == -1 {
			pos += file.Length
			log.Debugf("skip file %s", fp)
			continue
		}
		var f *os.File
		if t.meta.IsSingleFile() {
			f, err = file.Path.Open(t.st.DataDir)
			at = int64(offset)
		} else {
			f, err = file.Path.Open(t.FilePath())
			if at == -1 {
				if pos < offset {
					at = int64(offset - pos)
				} else {
					at = int64(offset)
				}
			}
		}
		log.Debugf("open %s", f.Name())
		if err == nil {
			var n int
			log.Debugf("GetPiece() %s pos=%d offset=%d at=%d left=%d", fp, pos, offset, at, len(readbuf))
			n, err = f.ReadAt(readbuf, at)
			log.Debugf("Read %d", n)
			if err == io.EOF {
				at = 0
				err = nil
			}
			if err == nil {
				pos += uint64(n)
				readbuf = readbuf[n:]
			} else {
				log.Warnf("GetPiece(): error reading %s, %s, read %d", fp, err, n)
			}
			f.Close()
		}
		if err == nil && len(readbuf) == 0 {
			p = pc
			break
		}
	}
	return
}

func (t *fsTorrent) checkPiece(pc *common.PieceData) (err error) {
	if pc == nil || !t.meta.Info.CheckPiece(pc) {
		if pc == nil {
			log.Errorf("tried to store nil piece for %s", t.Name())
		}
		err = common.ErrInvalidPiece
	}
	return
}

func (t *fsTorrent) PutPiece(pc *common.PieceData) (err error) {

	err = t.checkPiece(pc)
	if err == nil {

		files := t.meta.Info.GetFiles()
		sz := t.meta.Info.PieceLength
		offset := uint64(pc.Index * sz)
		pos := uint64(0)
		buf := pc.Data[:]
		at := int64(-1)
		for _, file := range files {
			if pos+file.Length < offset {
				pos += file.Length
				continue
			}
			if len(buf) == 0 {
				break
			}
			fp := file.Path.FilePath()
			var f *os.File
			f, err = file.Path.Open(t.st.DataDir)
			if err == nil {
				defer f.Close()
				var n int
				left := uint64(len(buf))
				if at < 0 {
					if pos > offset {
						at = int64(pos - offset)
					} else {
						at = int64(offset)
					}
				} else {
					at = int64(pos)
				}
				if left < file.Length {
					// entire file
					n, err = f.WriteAt(buf, at)
					log.Debugf("write full %d at %d", n, at)
				} else if left > 0 {
					// part of the file
					n, err = f.WriteAt(buf[:file.Length], at)
					log.Debugf("write part %d at %d", n, at)
				} else {
					// done
					break
				}
				log.Debugf("PutPiece() %s %d %d", fp, pos, left)

				if err == io.EOF {
					err = nil
				}
				if err == nil && n > 0 {
					pos += uint64(n)
					buf = buf[n:]
				} else {
					log.Warnf("PutPiece(): error writing %s, %s, write %d", fp, err, n)
				}
			}
		}
	}
	return
}

func (t *fsTorrent) putPieceOld(pc *common.PieceData) error {

	// check integrity
	err := t.checkPiece(pc)
	if err != nil {
		return err
	}
	sz := uint64(t.meta.Info.PieceLength)
	if t.meta.IsSingleFile() {
		f, err := os.OpenFile(t.FilePath(), os.O_WRONLY, 0640)
		if err != nil {
			log.Errorf("failed to open %s: %s", t.FilePath(), err)
			return err
		}
		idx := int64(pc.Index) * int64(sz)
		_, err = f.WriteAt(pc.Data, idx)
		f.Close()
	} else {
		idx := uint64(0)
		cur := uint64(0)
		left := uint64(sz)
		i := t.MetaInfo()
		np := uint64(i.Info.NumPieces())
		if uint64(pc.Index)+1 == np {
			left -= (sz * np) - i.TotalSize()
		}
		pieceOff := sz * uint64(pc.Index)
		for _, info := range t.meta.Info.Files {
			if info.Length+cur >= pieceOff {
				fpath := filepath.Join(t.FilePath(), info.Path.FilePath())
				f, err := os.OpenFile(fpath, os.O_WRONLY, 0640)
				if err == nil {
					defer f.Close()
					if info.Length <= left {
						_, err = f.Write(pc.Data[idx : idx+info.Length])
						//err = util.WriteFull(f, pc.Data[idx:idx+info.Length])
						idx += info.Length
						left -= info.Length
						cur += info.Length
						if err != nil {
							log.Errorf("Failed to write %s: %s", fpath, err)
							return err
						}
						continue
					} else {
						var n int
						n, err = f.WriteAt(pc.Data[idx:idx+left], int64(pieceOff-cur)-int64(left))
						// err = util.WriteFull(f, pc.Data[idx:left])
						if err != nil {
							log.Errorf("Failed to write %s: %s", fpath, err)
							return err
						}
						idx += uint64(n)
						break
					}
				} else {
					log.Errorf("Failed to open %s: %s", fpath, err)
					return err
				}
			}
			cur += info.Length
		}
	}
	// set bitfield
	t.bf.Set(pc.Index)
	return nil
}

func (t *fsTorrent) VerifyAll(fresh bool) (err error) {
	t.bfmtx.Lock()
	check := t.st.FindBitfield(t.ih)
	if check == nil {
		// no stored bitfield
		log.Infof("no bitfield for %s", t.Name())
		check = bittorrent.NewBitfield(t.meta.Info.NumPieces(), nil).Inverted()
		if fresh {
			var has *bittorrent.Bitfield
			has, err = t.verifyBitfield(check, false)
			t.st.flushBitfield(t.ih, has)
			t.bfmtx.Unlock()
			return
		}
	}
	// verify
	log.Infof("verify local data for %s", t.Name())
	t.bf, err = t.verifyBitfield(check, true)
	if err == nil {
		if t.bf.Equals(check) {
			log.Infof("%s check okay", t.Name())
		} else {
			log.Warnf("%s has miss matched data", t.Name())
		}
	} else {
		t.bfmtx.Unlock()
		return
	}
	t.bfmtx.Unlock()
	err = t.Flush()
	return
}

// verifyBitfield verifies a all pieces given by a bitfield
func (t *fsTorrent) verifyBitfield(bf *bittorrent.Bitfield, warn bool) (has *bittorrent.Bitfield, err error) {
	np := t.meta.Info.NumPieces()
	has = bittorrent.NewBitfield(np, nil)
	sz := uint64(t.meta.Info.PieceLength)
	tl := t.meta.TotalSize()
	idx := uint32(0)
	for idx < np {
		l := t.meta.Info.PieceLength
		if idx == np-1 {
			l -= uint32((uint64(np) * sz) - tl)
		}
		if bf.Has(idx) {
			var pc *common.PieceData
			pc, err = t.GetPiece(&common.PieceRequest{
				Index:  idx,
				Length: l,
			})
			if err == nil {
				err = t.checkPiece(pc)
				if err == nil {
					has.Set(idx)
				} else if warn {
					log.Warnf("piece %d failed check for %s: %s", idx, t.Name(), err)
				}
				err = nil
			} else {
				log.Errorf("failed to get piece %d for %s: %s", idx, t.Name(), err)
			}
		}
		idx++
		log.Debugf("piece %d of %d", idx, np)
	}
	return
}

func (t *fsTorrent) Flush() error {
	log.Debugf("flush bitfield for %s", t.ih.Hex())
	bf := t.Bitfield()
	return t.st.flushBitfield(t.ih, bf)
}

// filesystem based torrent storage
type FsStorage struct {
	// directory for downloaded data
	DataDir string
	// directory for torrent seed data
	MetaDir string
}

func (st *FsStorage) flushBitfield(ih common.Infohash, bf *bittorrent.Bitfield) (err error) {
	fname := st.bitfieldFilename(ih)
	var f *os.File
	f, err = os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, 0600)
	if err == nil {
		err = bf.BEncode(f)
		f.Close()
	}
	return
}

func (st *FsStorage) Init() (err error) {
	log.Info("Ensure filesystem storage")
	if st.DataDir == "" || st.MetaDir == "" {
		err = errors.New("bad FsStorage parameters")
		return
	}
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

func (st *FsStorage) CreateNewBitfield(ih common.Infohash, bits uint32) {
	fname := st.bitfieldFilename(ih)
	bf := bittorrent.NewBitfield(bits, nil)
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY, 0600)
	if err == nil {
		bf.BEncode(f)
		f.Close()
	}
}

func (st *FsStorage) OpenTorrent(info *metainfo.TorrentFile) (t Torrent, err error) {
	basepath := filepath.Join(st.DataDir, info.TorrentName())
	if !info.IsSingleFile() {
		// create directory
		os.Mkdir(basepath, 0700)
	}

	ih := info.Infohash()
	metapath := filepath.Join(st.MetaDir, ih.Hex()+".torrent")
	_, err = os.Stat(metapath)

	if os.IsNotExist(err) {
		// put meta info down onto filesystem
		var f *os.File
		f, err = os.OpenFile(metapath, os.O_CREATE|os.O_WRONLY, 0600)
		if err == nil {
			err = info.BEncode(f)
			f.Close()
		}
	}

	if err == nil {
		ft := &fsTorrent{
			st:   st,
			meta: info,
			ih:   ih,
		}
		log.Debugf("allocate space for %s", ft.Name())
		err = ft.Allocate()
		if err != nil {
			t = nil
			return
		}
		t = ft
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
			log.Warnf("error checking torrent file %s: %s", m, err)
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
