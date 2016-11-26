package metainfo

import (
	"crypto/sha1"
	"io"
	"path/filepath"
	"xd/lib/common"
	"github.com/zeebo/bencode"
)


type FilePath []string

// get filepath
func (f FilePath) FilePath() string {
	return filepath.Join(f...)
}

type FileInfo struct {
	// length of file
	Length int64 `bencode:length`
	// relative path of file
	Path FilePath `bencode:name`
}

// info section of torrent file
type Info struct {
	// length of pices in bytes
	PieceLength int64 `bencode:"piece length"`
	// piece data
	Pieces []byte `bencode:"pieces"`
	// name of root file
	Path string `bencode:"name"`
	// file metadata
	Files []FileInfo `bencode:"files,omitemtpy"`
	// private torrent
	Private int64 `bencode:"private"`
	// length of file in signle file mode
	Length int64 `bencode:"length,omitempty"`
}

// get total size of files from torrent info section
func (i Info) TotalSize() int64 {
	return int64(len(i.Pieces)) * i.PieceLength
}

// a torrent file
type TorrentFile struct {
	Info Info `bencode:"info"`
	Announce string `bencode:"announce"`
	AnnounceList []string `bencode:"announce-list"`
	Created int64   `bencode:"created"`
	Comment string  `bencode:"comment"`
	CreatedBy int64 `bencode:"created by"`
	Encoding string `bencode:"encoding"`
}

func (tf *TorrentFile) GetAllAnnounceURLS() (l []string) {
	if len(tf.Announce) > 0 {
		l = append(l, tf.Announce)
	}
	for _, a := range tf.AnnounceList {
		if len(a) > 0 {
			l = append(l, a)
		}
	}
	return
}

func (tf *TorrentFile) TorrentName() string {
	return tf.Info.Path
}

// calculate infohash
func (tf *TorrentFile) Infohash() (ih common.Infohash) {
	h := sha1.New()
	enc := bencode.NewEncoder(h)
	enc.Encode(tf.Info)
	d := h.Sum(nil)
	copy(ih[:], d[:])
	return
}

// return true if this torrent is for a single file
func (tf *TorrentFile) IsSingleFile() bool {
	return tf.Info.Length > 0
}

// bencode this file via an io.Writer
func (tf *TorrentFile) BEncode(w io.Writer) (err error) {
	enc := bencode.NewEncoder(w)
	err = enc.Encode(tf)
	return
}

// load from an io.Reader
func (tf *TorrentFile) BDecode(r io.Reader) (err error) {
	dec := bencode.NewDecoder(r)
	err = dec.Decode(tf)
	return
}
