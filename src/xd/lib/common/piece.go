package common

import "errors"

// PieceData is a bittorrent piece response
type PieceData struct {
	Index uint32
	Begin uint32
	Data  []byte
}

// PieceRequest is a request for a bittorrent piece
type PieceRequest struct {
	Index  uint32
	Begin  uint32
	Length uint32
}

// ErrInvalidPiece is an error for when a piece has invalid sha1sum
var ErrInvalidPiece = errors.New("invalid piece")
