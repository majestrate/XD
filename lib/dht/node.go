package dht

type Node interface {
	SendLowLevel([]byte) error
	ID() string
}
