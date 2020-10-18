package mktorrent

import (
	"crypto/sha1"
	"errors"
	"io"
	"path/filepath"
	"github.com/majestrate/XD/lib/fs"
	"github.com/majestrate/XD/lib/metainfo"
)

func mkTorrentSingle(f fs.Driver, fpath string, pieceLength uint32) (*metainfo.TorrentFile, error) {
	var info metainfo.Info

	info.PieceLength = pieceLength
	info.Path = filepath.Base(fpath)

	r, err := f.OpenFileReadOnly(fpath)
	if err != nil {
		return nil, err
	}
	buff := make([]byte, info.PieceLength)
	for {
		n, err := io.ReadFull(r, buff)
		if err == io.ErrUnexpectedEOF {
			err = nil
			d := sha1.Sum(buff[0:n])
			info.Pieces = append(info.Pieces, d[:]...)
			info.Length += uint64(n)
			break
		} else if err == nil {
			d := sha1.Sum(buff)
			info.Pieces = append(info.Pieces, d[:]...)
			info.Length += uint64(n)
		} else {
			return nil, err
		}
	}

	return &metainfo.TorrentFile{
		Info: info,
	}, nil
}

func mkTorrentDir(f fs.Driver, fpath string, pieceLength uint32) (*metainfo.TorrentFile, error) {
	return nil, errors.New("not implemented")
}

func MakeTorrent(f fs.Driver, fpath string, pieceLength uint32) (*metainfo.TorrentFile, error) {
	st, err := f.Stat(fpath)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return mkTorrentDir(f, fpath, pieceLength)
	}
	return mkTorrentSingle(f, fpath, pieceLength)
}
