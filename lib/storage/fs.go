package storage

import (
	"errors"
	"io"
	"github.com/majestrate/XD/lib/bittorrent"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/fs"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/metainfo"
	"github.com/majestrate/XD/lib/stats"
	"github.com/majestrate/XD/lib/sync"
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
	// base directory
	dir string
	// storage access mutex
	access sync.Mutex
	// set to true when we are doing a deep check
	checking bool
	// set to true when we did a deep check
	seeding bool
	// seeding mutex
	seedAccess sync.Mutex
}

func (t *fsTorrent) DownloadDir() string {
	return t.dir
}

func (t *fsTorrent) Delete() (err error) {
	err = t.st.FS.RemoveAll(t.st.metainfoFilename(t.ih))
	if err == nil {
		err = t.st.FS.RemoveAll(t.st.bitfieldFilename(t.ih))
		if err == nil {
			err = t.st.FS.RemoveAll(t.FilePath())
		}
	}
	return
}

func (t *fsTorrent) MoveTo(other string) (err error) {
	t.access.Lock()
	err = t.st.FS.EnsureDir(other)
	if err == nil {
		multifile := !t.MetaInfo().IsSingleFile()
		files := t.MetaInfo().Info.GetFiles()
		for _, file := range files {
			root := ""
			if multifile {
				root = t.MetaInfo().Info.Path
			}
			oldpath := file.Path.FilePath(t.st.FS.Join(t.dir, root))
			newpath := file.Path.FilePath(t.st.FS.Join(other, root))
			log.Debugf("move %s -> %s", oldpath, newpath)
			err = t.st.FS.Move(oldpath, newpath)
			if err != nil {
				break
			}
		}
	}
	s := t.st.getSettings(t.ih)
	s.Put("dir", other)
	t.st.putSettings(t.ih, s)
	t.dir = other
	t.access.Unlock()
	return
}

func (t *fsTorrent) AllocateFile(f metainfo.FileInfo) (err error) {
	fname := t.st.FS.Join(t.FilePath(), f.Path.FilePath(""))
	err = t.st.FS.EnsureFile(fname, f.Length)
	return
}

func (t *fsTorrent) Allocate() (err error) {
	if t.meta.IsSingleFile() {
		log.Debugf("file is %d bytes", t.meta.Info.Length)
		err = t.st.FS.EnsureFile(t.FilePath(), t.meta.Info.Length)
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

func (t *fsTorrent) openfileRead(i metainfo.FileInfo) (f fs.ReadFile, err error) {
	var fname string
	if t.meta.IsSingleFile() {
		fname = t.FilePath()
	} else {
		fname = t.st.FS.Join(t.FilePath(), i.Path.FilePath(""))
	}
	f, err = t.st.FS.OpenFileReadOnly(fname)
	return
}

func (t *fsTorrent) openfileWrite(i metainfo.FileInfo) (f fs.WriteFile, err error) {
	var fname string
	if t.meta.IsSingleFile() {
		fname = t.FilePath()
	} else {
		fname = t.st.FS.Join(t.FilePath(), i.Path.FilePath(""))
	}
	f, err = t.st.FS.OpenFileWriteOnly(fname)
	return
}

func (t *fsTorrent) readFileAt(fi metainfo.FileInfo, b []byte, off int64) (n int, err error) {

	// from github.com/anacrolix/torrent
	var f fs.ReadFile
	f, err = t.openfileRead(fi)
	fil := int64(fi.Length)
	// Limit the read to within the expected bounds of this file.
	if int64(len(b)) > fil-off {
		b = b[:fil-off]
	}
	for off < fil && len(b) != 0 {
		n1, err1 := f.ReadAt(b, off)
		b = b[n1:]
		n += n1
		off += int64(n1)
		if n1 == 0 {
			err = err1
			break
		}
	}
	if f != nil {
		f.Close()
	}
	return
}

func (t *fsTorrent) ReadAt(b []byte, off int64) (n int, err error) {

	files := t.meta.Info.GetFiles()
	// from github.com/anacrolix/torrent
	for _, fi := range files {
		fil := int64(fi.Length)
		for off < fil {
			n1, err1 := t.readFileAt(fi, b, off)
			n += n1
			off += int64(n1)
			b = b[n1:]

			if len(b) == 0 {
				// Got what we need.
				return
			}
			if n1 != 0 {
				// Made progress.
				continue
			}
			err = err1
			if err == io.EOF {
				// Lies.
				err = io.ErrUnexpectedEOF
			}
			return
		}
		off -= fil
	}
	err = io.EOF
	return
}

func (t *fsTorrent) WriteAt(p []byte, off int64) (n int, err error) {

	// from github.com/anacrolix/torrent
	for _, fi := range t.meta.Info.GetFiles() {
		fil := int64(fi.Length)
		if off >= fil {
			off -= fil
			continue
		}
		n1 := len(p)
		if int64(n1) > fil-off {
			n1 = int(fil - off)
		}
		var f fs.WriteFile
		f, err = t.openfileWrite(fi)
		if err != nil {
			return
		}
		n1, err = f.WriteAt(p[:n1], off)
		f.Sync()
		f.Close()
		if err == io.ErrUnexpectedEOF {
			err = nil
		}
		if err != nil {
			return
		}
		n += n1
		off = 0
		p = p[n1:]
		if len(p) == 0 {
			break
		}
	}
	return
}

func (t *fsTorrent) Bitfield() *bittorrent.Bitfield {
	t.bfmtx.Lock()
	t.ensureBitfield()
	t.bfmtx.Unlock()
	return t.bf
}

func (t *fsTorrent) ensureBitfield() {
	if t.meta == nil {
		return
	}
	if t.bf == nil {
		if !t.st.HasBitfield(t.ih) {
			// we have no pieces
			t.st.CreateNewBitfield(t.ih, t.meta.Info.NumPieces())
		}
		t.bf = t.st.FindBitfield(t.ih)
	}
}

func (t *fsTorrent) DownloadRemaining() (r uint64) {
	if t.meta == nil {
		return
	}
	bf := t.Bitfield()
	have := uint64(bf.CountSet()) * uint64(t.meta.Info.PieceLength)
	r = t.meta.TotalSize() - have
	return
}

func (t *fsTorrent) MetaInfo() *metainfo.TorrentFile {
	return t.meta
}

func (t *fsTorrent) Name() string {
	if t.meta == nil {
		return t.Infohash().Hex()
	}
	return t.meta.TorrentName()
}

func (t *fsTorrent) Infohash() (ih common.Infohash) {
	copy(ih[:], t.ih[:])
	return
}

func (t *fsTorrent) FilePath() string {
	if t.meta == nil {
		return ""
	}
	return t.st.FS.Join(t.dir, t.meta.Info.Path)

}

func (t *fsTorrent) PutInfo(info metainfo.Info) (err error) {
	if t.meta == nil {
		meta := &metainfo.TorrentFile{
			Info: info,
		}
		ih := meta.Infohash()
		if !t.ih.Equal(ih) {
			err = ErrMetaInfoMissmatch
			return
		}
		t.access.Lock()
		t.meta = meta
		metapath := t.st.metainfoFilename(ih)
		var f fs.WriteFile
		f, err = t.st.FS.OpenFileWriteOnly(metapath)
		if err == nil {
			err = t.meta.BEncode(f)
			f.Close()
			if err == nil {
				log.Debugf("allocate room for %s", t.Name())
				err = t.Allocate()
			}
		}
		t.access.Unlock()
	}
	return
}

func (t *fsTorrent) GetPiece(r common.PieceRequest, pc *common.PieceData) (err error) {
	t.access.Lock()
	sz := t.meta.Info.PieceLength
	offset := int64(r.Begin) + (int64(sz) * int64(r.Index))
	pc.Data = make([]byte, r.Length)
	log.Debugf("get piece %d offset=%d len=%d", r.Index, r.Begin, r.Length)
	if t.st.pooledIO() {
		iop := readIOP{
			offset:    offset,
			r:         t,
			data:      pc.Data,
			replyChnl: make(chan iopResult),
		}
		t.st.ioChan <- &iop
		res := <-iop.replyChnl
		err = res.err
	} else {
		_, err = t.ReadAt(pc.Data, offset)
	}
	if err == nil {
		pc.Index = r.Index
		pc.Begin = r.Begin
	}
	t.access.Unlock()
	return
}

func (t *fsTorrent) VerifyPiece(idx uint32) (err error) {
	l := t.meta.LengthOfPiece(idx)
	r := common.PieceRequest{
		Index:  idx,
		Length: l,
	}
	var pc common.PieceData
	pc.Data = make([]byte, l)
	pc.Index = idx
	err = t.GetPiece(r, &pc)
	if err == nil {
		if t.meta.Info.CheckPiece(&pc) {
			t.bf.Set(idx)
		} else {
			t.bf.Unset(idx)
			err = common.ErrInvalidPiece
		}
	}
	return
}

func (t *fsTorrent) VerifyAll() (err error) {
	if t.meta == nil {
		err = ErrNoMetaInfo
		return
	}
	t.bfmtx.Lock()
	t.checking = true
	log.Infof("checking local data for %s", t.Name())
	t.ensureBitfield()
	info := t.MetaInfo().Info
	sz := info.NumPieces()
	idx := uint32(0)
	for idx < sz {
		err = t.VerifyPiece(uint32(idx))
		if err == common.ErrInvalidPiece {
			err = nil
		} else if err != nil {
			log.Errorf("failed to check piece %d: %s", idx, err.Error())
		}
		idx++
	}
	t.seeding = t.bf.Completed()
	t.bfmtx.Unlock()
	log.Infof("local data check done for %s", t.Name())
	err = t.Flush()
	t.checking = false
	return
}

func (t *fsTorrent) PutChunk(d *common.PieceData) (err error) {
	err = t.putChunk(d.Index, d.Begin, d.Data)
	return
}

func (t *fsTorrent) putChunk(idx, offset uint32, data []byte) (err error) {
	if t.meta == nil {
		err = ErrNoMetaInfo
		return
	}
	t.access.Lock()
	sz := int64(t.meta.Info.PieceLength)
	off := (sz * int64(idx)) + int64(offset)
	log.Debugf("put chunk idx=%d off=%d globaloff=%d len=%d", idx, offset, off, len(data))
	if t.st.pooledIO() {
		iop := writeIOP{
			data:      data,
			offset:    off,
			w:         t,
			replyChnl: make(chan iopResult),
		}
		t.st.ioChan <- &iop
		res := <-iop.replyChnl
		err = res.err
	} else {
		_, err = t.WriteAt(data, off)
	}
	t.access.Unlock()
	return
}

func (t *fsTorrent) Flush() error {
	if t.meta == nil {
		return ErrNoMetaInfo
	}
	log.Debugf("flush bitfield for %s", t.ih.Hex())
	bf := t.Bitfield()
	return t.st.flushBitfield(t.ih, bf)
}

func (t *fsTorrent) Close() error {
	return t.Flush()
}

func (t *fsTorrent) SaveStats(s *stats.Tracker) (err error) {
	err = t.st.saveStatsForTorrent(t.ih, s)
	return
}

func (t *fsTorrent) Checking() bool {
	return t.checking
}

func (t *fsTorrent) FileList() (flist []string) {
	if t.meta != nil {
		files := t.meta.Info.GetFiles()
		flist = make([]string, len(files))
		for idx, f := range files {
			flist[idx] = f.Path.FilePath(t.dir)
		}
	}
	return
}

func (t *fsTorrent) Seed() (bool, error) {
	err := t.doSeed()
	return t.seeding, err
}

func (t *fsTorrent) doSeed() (err error) {
	t.seedAccess.Lock()
	defer t.seedAccess.Unlock()
	if t.seeding {
		t.seeding = true
		return
	}
	err = t.VerifyAll()
	if err == nil {
		if t.dir != t.st.SeedingDir {
			log.Infof("Moving downloaded data to %s", t.st.SeedingDir)
			err = t.MoveTo(t.st.SeedingDir)
		}
		t.seeding = err == nil
	} else if err == common.ErrInvalidPiece {
		log.Error("invalid pieces will redownload")
		err = nil
		t.seeding = false
	}
	return
}

type IOP interface {
	RunIOP()
}

type iopResult struct {
	err error
	n   int
}

type readIOP struct {
	data      []byte
	offset    int64
	r         io.ReaderAt
	replyChnl chan iopResult
}

func (iop *readIOP) RunIOP() {
	n, err := iop.r.ReadAt(iop.data, iop.offset)
	iop.replyChnl <- iopResult{n: n, err: err}
}

type writeIOP struct {
	data      []byte
	offset    int64
	w         io.WriterAt
	replyChnl chan iopResult
}

func (iop *writeIOP) RunIOP() {
	n, err := iop.w.WriteAt(iop.data, iop.offset)
	iop.replyChnl <- iopResult{n: n, err: err}
}

// filesystem based torrent storage
type FsStorage struct {
	// directory for seeding data
	SeedingDir string
	// directory for downloaded data
	DataDir string
	// directory for torrent seed data
	MetaDir string
	// filesystem driver
	FS fs.Driver
	// number of io worker threads
	Workers int
	// IOP channel buffer size
	IOPBufferSize int
	// buffered io channel
	ioChan chan IOP
}

func (st *FsStorage) Run() {
	n := st.Workers
	if n <= 0 {
		st.ioChan = nil
	} else {
		workers := n
		buff := st.IOPBufferSize
		if buff <= 0 {
			buff = 128
		}
		var wg sync.WaitGroup
		st.ioChan = make(chan IOP, buff)
		for workers > 0 {
			go func() {
				wg.Add(1)
				for {
					iop := <-st.ioChan
					if iop == nil {
						wg.Add(-1)
						return
					}
					iop.RunIOP()
				}
			}()
			workers--
		}
		wg.Wait()
	}
}

func (st *FsStorage) Close() (err error) {
	if st.pooledIO() {
		workers := st.Workers
		for workers > 0 {
			st.ioChan <- nil
			workers--
		}
	}
	err = st.FS.Close()
	return
}

func (st *FsStorage) flushBitfield(ih common.Infohash, bf *bittorrent.Bitfield) (err error) {
	fname := st.bitfieldFilename(ih)
	var f fs.WriteFile
	f, err = st.FS.OpenFileWriteOnly(fname)
	if err == nil {
		err = bf.BEncode(f)
		f.Close()
	}
	return
}

func (st *FsStorage) Init() (err error) {
	log.Info("Ensure filesystem storage")
	err = st.FS.Open()
	if err != nil {
		return
	}
	if st.DataDir == "" || st.MetaDir == "" {
		err = errors.New("bad FsStorage parameters")
		return
	}
	err = st.FS.EnsureDir(st.DataDir)
	if err == nil {
		err = st.FS.EnsureDir(st.MetaDir)
	}
	if err == nil {
		err = st.FS.EnsureDir(st.SeedingDir)
	}
	return
}

func (st *FsStorage) FindBitfield(ih common.Infohash) (bf *bittorrent.Bitfield) {
	fpath := st.bitfieldFilename(ih)
	f, err := st.FS.OpenFileReadOnly(fpath)
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
	return st.FS.Join(st.MetaDir, ih.Hex()+".bitfield")
}

func (st *FsStorage) HasBitfield(ih common.Infohash) bool {
	return st.FS.FileExists(st.bitfieldFilename(ih))
}

func (st *FsStorage) CreateNewBitfield(ih common.Infohash, bits uint32) {
	fname := st.bitfieldFilename(ih)
	bf := bittorrent.NewBitfield(bits, nil)
	f, err := st.FS.OpenFileWriteOnly(fname)
	if err == nil {
		bf.BEncode(f)
		f.Close()
	}
}

func (st *FsStorage) metainfoFilename(ih common.Infohash) string {
	return st.FS.Join(st.MetaDir, ih.Hex()+".torrent")
}

func (st *FsStorage) statsFilename(ih common.Infohash) string {
	return st.FS.Join(st.MetaDir, ih.Hex()+".stats")
}

func (st *FsStorage) settingsFilename(ih common.Infohash) string {
	return st.FS.Join(st.MetaDir, ih.Hex()+".settings")
}

func (st *FsStorage) saveStatsForTorrent(ih common.Infohash, s *stats.Tracker) (err error) {
	var f fs.WriteFile
	f, err = st.FS.OpenFileWriteOnly(st.statsFilename(ih))
	if err == nil {
		err = s.BEncode(f)
		f.Close()
	}
	return
}

func (st *FsStorage) EmptyTorrent(ih common.Infohash) (t Torrent) {
	t = &fsTorrent{
		dir: st.DataDir,
		st:  st,
		ih:  ih,
	}
	return
}

func (st *FsStorage) OpenTorrent(info *metainfo.TorrentFile) (t Torrent, err error) {
	t, err = st.openTorrent(info, st.DataDir)
	return
}

func (st *FsStorage) openTorrent(info *metainfo.TorrentFile, rootpath string) (t Torrent, err error) {
	basepath := st.FS.Join(rootpath, info.TorrentName())
	if !info.IsSingleFile() {
		// create directory
		st.FS.EnsureDir(basepath)
	}

	ih := info.Infohash()
	metapath := st.metainfoFilename(ih)
	if !st.FS.FileExists(metapath) {
		// put meta info down onto filesystem
		var f fs.WriteFile
		f, err = st.FS.OpenFileWriteOnly(metapath)
		if err == nil {
			info.BEncode(f)
			f.Close()
		}
	}

	if err == nil {
		ft := &fsTorrent{
			dir:  rootpath,
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

// return true if we are using pooled io
func (st *FsStorage) pooledIO() bool {
	return st.ioChan != nil
}

func (st *FsStorage) initSettings(i common.Infohash) {
	s := createSettings()
	s.Put("dir", st.DataDir)
	st.putSettings(i, s)
}

func (st *FsStorage) putSettings(i common.Infohash, s fsSettings) {
	f, _ := st.FS.OpenFileWriteOnly(st.settingsFilename(i))
	if f != nil {
		s.BEncode(f)
		f.Close()
	}
}

func (st *FsStorage) getSettings(i common.Infohash) (s fsSettings) {
	s = createSettings()
	if !st.FS.FileExists(st.settingsFilename(i)) {
		st.initSettings(i)
	}
	f, _ := st.FS.OpenFileReadOnly(st.settingsFilename(i))
	if f != nil {
		s.BDecode(f)
		f.Close()
	}
	return
}

func (st *FsStorage) OpenAllTorrents() (torrents []Torrent, err error) {
	var matches []string
	matches, err = st.FS.Glob(st.FS.Join(st.MetaDir, "*.torrent"))
	for _, m := range matches {
		var t Torrent
		var f fs.ReadFile
		tf := new(metainfo.TorrentFile)
		f, err = st.FS.OpenFileReadOnly(m)
		if err == nil {
			err = tf.BDecode(f)
			f.Close()
		}
		if err == nil {
			s := st.getSettings(tf.Infohash())
			path := s.Get("dir", st.DataDir)
			t, err = st.openTorrent(tf, path)
		}
		if t != nil {
			torrents = append(torrents, t)
		}
	}
	return
}

func (st *FsStorage) PollNewTorrents() (torrents []Torrent) {
	matches, _ := st.FS.Glob(st.FS.Join(st.DataDir, "*.torrent"))
	for _, m := range matches {
		var t Torrent
		tf := new(metainfo.TorrentFile)
		f, err := st.FS.OpenFileReadOnly(m)
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
