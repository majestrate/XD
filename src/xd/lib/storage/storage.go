package storage

import (
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/metainfo"
)

// storage session for 1 torrent
type Torrent interface {

	// allocate all files for download
	Allocate() error

	// verify all piece data
	VerifyAll(fresh bool) error

	// put a downloaded piece into the storage
	PutPiece(p *common.PieceData) error

	// get a piece from storage
	// returns nil if we don't have the data
	GetPiece(r *common.PieceRequest) (*common.PieceData, error)

	// visit a piece from storage
	VisitPiece(r *common.PieceRequest, f func(*common.PieceData)) error

	// verify a piece by index
	VerifyPiece(idx uint32) error

	// get metainfo
	MetaInfo() *metainfo.TorrentFile

	// get infohash
	Infohash() common.Infohash

	// get bitfield, if cached return cache otherwise compute and cache
	Bitfield() *bittorrent.Bitfield

	// get number of bytes remaining we need to download
	DownloadRemaining() uint64

	// flush bitfield to disk
	Flush() error

	// get name of this torrent
	Name() string
}

// torrent storage driver
type Storage interface {

	// open a storage session for a torrent
	// does not verify any piece data
	OpenTorrent(info *metainfo.TorrentFile) (Torrent, error)

	// open all torrents tracked by this storage
	// does not verify any piece data
	OpenAllTorrents() ([]Torrent, error)

	// intialize backend
	Init() error

	// returns nil if we have no new torrents added from backend
	// returns next new torrents added to storage
	PollNewTorrents() []Torrent
}
