package common

import "errors"

// bittorrent data piece
type Piece struct {
	Index int64
	Data  []byte
}

// error for when piece has invalid sha1sum
var ErrInvalidPiece = errors.New("invalid piece")
