package storage

import (
	"errors"
	"github.com/majestrate/XD/lib/bittorrent"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/metainfo"
	"github.com/majestrate/XD/lib/stats"
)

var ErrNoMetaInfo = errors.New("no torrent file")
var ErrMetaInfoMissmatch = errors.New("torrent infohash does not match")

// storage session for 1 torrent
type Torrent interface {

	// allocate all files for download
	Allocate() error

	// verify all piece data
	VerifyAll() error

	// return true if we are currently doing a deep check
	Checking() bool

	// put a chunk of data 
	PutChunk(pc *common.PieceData) error

	// visit a piece from storage
	GetPiece(r common.PieceRequest, pc *common.PieceData) error

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

	// delete all files and metadata for this torrent
	Delete() error

	// save torrent stats
	SaveStats(s *stats.Tracker) error

	// get a list of files for this torrent
	// returns absolute path of all downloaded files
	FileList() []string

	// move data files to other directory, blocks for a LONG time
	MoveTo(other string) error

	// verify data and move to seeding directory
	Seed() (bool, error)

	// set metainfo for empty torrent
	PutInfo(info metainfo.Info) error

	// get directory for data files
	DownloadDir() string
}

// torrent storage driver
type Storage interface {

	// Close and flush storage backend
	Close() error

	// create a torrent with no meta info
	EmptyTorrent(ih common.Infohash) Torrent

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

	// run mainloop
	Run()
}
