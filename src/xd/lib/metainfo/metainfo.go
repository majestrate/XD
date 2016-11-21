package metainfo

import (
	"path/filepath"
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

