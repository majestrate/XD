package dht

type Payload interface {
	Serialize
	Process(ctx Context, ch chan *Message) // reply to this message by sending the reply down ch
}
