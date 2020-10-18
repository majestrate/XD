package dht

const mFindNode = "find_node"
const mGetPeers = "get_peers"
const mAnnouncePeer = "announce_peer"

const kQuery = "q"
const kResponse = "r"
const kError = "e"

const vID = "id"
const vTarget = "target"
const vNodes = "nodes"

type Message struct {
	Query string                 `bencode:"q",omitempty`
	TID   string                 `bencode:"t"`
	Reply string                 `bencode:"y"`
	Err   *Error                 `bencode:"e",omitempty`
	Args  map[string]interface{} `bencode:"a",omitempty`
}

func (m *Message) IsError() bool {
	return m.Reply == kError
}

// NewError generates a new error reply message
func NewError(txid string, code int, errMsg string) *Message {
	return &Message{
		TID:   txid,
		Reply: kError,
		Err: &Error{
			Code:    int64(code),
			Message: errMsg,
		},
	}
}

func NewFindNodeRequest(txid, id, target string) *Message {
	return &Message{
		TID:   txid,
		Reply: kQuery,
		Query: mFindNode,
		Args: map[string]interface{}{
			vID:     id,
			vTarget: target,
		},
	}
}
