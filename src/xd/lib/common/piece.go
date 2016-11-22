package common

// bittorrent data piece
type Piece struct {
	Index int64
	Begin int64
	Data []byte
}
