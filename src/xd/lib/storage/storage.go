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
	VerifyAll() error

	// put a downloaded piece into the storage
	PutPiece(p *common.Piece)

	// get a piece from storage
	// returns nil if we don't have the data
	GetPiece(ind, off int64) *common.Piece

	// Verify Piece
	// returns error if verification failed otherwise nil
	Verify(piece int64) error

	// get metainfo
	MetaInfo() *metainfo.TorrentFile

	// get infohash
	Infohash() common.Infohash
	
	// get bitfield, if cached return cache otherwise compute and cache
	Bitfield() *bittorrent.Bitfield
}


// file storage driver
type Storage interface {
	
	// open a storage session for a torrent
	// does not verify any piece data
	OpenTorrent(info *metainfo.TorrentFile) (Torrent, error)

	// open all torrents tracked by this storage
	// does not verify any piece data
	OpenAllTorrents() ([]Torrent, error)
	
}
