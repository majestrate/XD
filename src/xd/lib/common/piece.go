package common

import (
	"bytes"
	"errors"
)

// PieceData is a bittorrent piece response
type PieceData struct {
	Index uint32
	Begin uint32
	Data  []byte
}

func (pc *PieceData) Equals(other *PieceData) bool {
	return pc != nil && other != nil && pc.Index == other.Index && pc.Begin == other.Begin && bytes.Equal(pc.Data, other.Data)
}

// PieceRequest is a request for a bittorrent piece
type PieceRequest struct {
	Index  uint32
	Begin  uint32
	Length uint32
}

// ErrInvalidPiece is an error for when a piece has invalid sha1sum
var ErrInvalidPiece = errors.New("invalid piece")
