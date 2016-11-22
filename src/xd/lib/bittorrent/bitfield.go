package bittorrent


type Bitfield []byte


func (bf Bitfield) ToWireMessage() *WireMessage {
	return NewWireMessage(5, bf[:])
}
