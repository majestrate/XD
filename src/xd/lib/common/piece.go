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

func (pc *PieceRequest) Copy(r *PieceRequest) {
	pc.Index = r.Index
	pc.Begin = r.Begin
	pc.Length = r.Length
}

func (pc PieceRequest) Cancel() WireMessage {
	return NewCancel(pc.Index, pc.Begin, pc.Length)
}

// ErrInvalidPiece is an error for when a piece has invalid sha1sum
var ErrInvalidPiece = errors.New("invalid piece")

// return true if piecedata matches this piece request
func (r *PieceRequest) Matches(d *PieceData) bool {
	return r.Length == uint32(len(d.Data)) && r.Begin == d.Begin && r.Index == d.Index
}

func (r *PieceRequest) Equals(other *PieceRequest) bool {
	return r.Index == other.Index && r.Length == other.Length && r.Begin == other.Begin
}
