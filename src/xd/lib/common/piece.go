package common

import "errors"

// bittorrent piece
type PieceData struct {
	Index uint32
	Begin uint32
	Data  []byte
}

// a bittorrent piece request
type PieceRequest struct {
	Index  uint32
	Begin  uint32
	Length uint32
}

// error for when piece has invalid sha1sum
var ErrInvalidPiece = errors.New("invalid piece")
