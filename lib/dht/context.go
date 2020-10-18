package dht

type Context interface {
	GetClosestNode(id []byte) Node
}
