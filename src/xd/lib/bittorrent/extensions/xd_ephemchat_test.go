package extensions

import (
	"github.com/zeebo/bencode"
	"testing"
	"xd/lib/crypto"
)

func TestMakeBCChat(t *testing.T) {
	testMsg := "test"
	sk := crypto.KeyGen()
	t.Logf("sk=%q", sk)
	chat := NewEphemChatBC("#test", *sk, nil, 0, testMsg)
	if chat.Verify() {
		data, _ := bencode.EncodeString(chat)
		t.Logf("message=%q", data)
	} else {
		t.Fail()
	}
}

func TestMakePMChat(t *testing.T) {
	testMsg := "test message"
	us := crypto.KeyGen()
	them := crypto.KeyGen()
	chat := NewEphemChatPM(them.ToPublic(), *us, nil, 0, testMsg)
	if chat.Verify() {
		data, _ := bencode.EncodeString(chat)
		dec, err := chat.Decrypt(*them)
		t.Logf("dec=%q, err=%q msg=%q", dec, err, data)
		if dec != testMsg {
			t.Fail()
		}
	} else {
		t.Fail()
	}
}
