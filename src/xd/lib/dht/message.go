package dht

const FindNode = "find_node"
const GetPeers = "get_peers"
const AnnouncePeer = "announce_peer"

const Query = "q"
const Response = "r"
const Error = "e"

const ID = "id"
const Target = "target"
const Nodes = "nodes"

type Message struct {
	Query string                 `bencode:"q",omitempty`
	TID   string                 `bencode:"t"`
	Reply string                 `bencode:"y"`
	Error []interface{}          `bencode:"e",omitempty`
	Args  map[string]interface{} `bencode:"a",omitempty`
}

func (m *Message) IsError() bool {
	return m.Reply == Error
}

// NewError generates a new error reply message
func NewError(txid string, code int, errMsg string) *Message {
	return &Message{
		TID:   txid,
		Reply: Error,
		Error: []interface{}{code, errMsg},
	}
}

func NewFindNodeRequest(txid, id, target string) *Message {
	return &Message{
		TID:   txid,
		Reply: Query,
		Query: FindNode,
		Args: map[string]interface{}{
			ID:     id,
			Target: target,
		},
	}
}

func NewFindNodeResponse(txid, id, nodes string) *Message {
	return &Message{
		TID:   txid,
		Reply: Response,
		Query: FindNode,
		Args: map[string]interface{}{
			ID:    id,
			Nodes: nodes,
		},
	}
}
