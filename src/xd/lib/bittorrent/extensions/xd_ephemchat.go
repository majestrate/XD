package extensions

import (
	"errors"
	"github.com/zeebo/bencode"
	"xd/lib/crypto"
)

const XDEphemChat = Extension("xd_ephemchat")

// 32 byte empty sig
const emptySig = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

type EphemChat struct {
	Sender  string `bencode:"a"`           // sender public key
	Target  string `bencode:"b,omitempty"` // broadcast target
	Recip   string `bencode:"c,omitempty"` // pm recipiant
	Message string `bencode:"m"`           // message
	Nounce  string `bencode:"n"`           // nounce
	Version int    `bencode:"v"`           // message version (currently 0)
	Sig     string `bencode:"z"`           // signature
}

type EphemChatReply struct {
	Nounce      string `bencode:"n"` // message nounce
	PowRequired int    `bencode:"p"` // POW required for delivery, if 0 delievery was good
	Version     int    `bencode:"v"` // message version (currently 0)
}

var ErrChatTooShort = errors.New("chat message is too short")
var ErrNounceTooShort = errors.New("chat nounce is too short")
var ErrInvalidChatPOW = errors.New("chat does not have enough POW")
var ErrInvalidChatPublicKey = errors.New("chat has invalid public key")

// verify message meets required pow
func (chat EphemChat) VerifyPOW(pow crypto.POW) bool {
	if pow == nil {
		return true
	}
	data, _ := bencode.EncodeBytes(chat)
	if data == nil {
		return false
	}
	return pow.VerifyWork(data)
}

// verify hmac of ciphertext and decrypt message
func (chat EphemChat) Decrypt(sk crypto.SecretKey) (decrypted string, err error) {
	if len(chat.Message) < 32 {
		err = ErrChatTooShort
		return
	}
	if len(chat.Nounce) < 32 {
		err = ErrNounceTooShort
		return
	}
	recip := sk.ToPublic()
	if recip.String() == chat.Recip {
		sender := crypto.NewPublicKey(chat.Sender)
		if sender == nil {
			err = ErrInvalidChatPublicKey
			return
		}
		n := []byte(chat.Nounce[0:32])
		shared := crypto.KeyExchange(recip, *sender, n)
		var hmac [32]byte
		copy(hmac[:], chat.Message[:32])
		err = crypto.VerifyHMac(chat.Message[32:], hmac, &shared)
		if err == nil {
			decrypted = crypto.Sym([]byte(chat.Message[32:]), n[0:24], &shared)
		}
	}
	return
}

// create new broadcast chat to target using secret key, pow and message using zero or more extra nounce bytes
func NewEphemChatBC(target string, sk crypto.SecretKey, pow crypto.POW, nounceExtra int, msg string) (chat EphemChat) {
	nounceSize := 32 + nounceExtra
	sender := sk.ToPublic()
	chat.Sender = sender.String()
	chat.Target = target
	chat.Nounce = crypto.RandStr(nounceSize)
	m := make([]byte, 32+len(msg))
	copy(m[32:], chat.Nounce[0:32])
	copy(m[:32], msg[:])
	chat.Message = msg
	chat.Sign(sk)
	for !chat.VerifyPOW(pow) {
		chat.Nounce = crypto.RandStr(nounceSize)
		copy(m[32:], chat.Nounce[0:32])
		chat.Sign(sk)
	}
	return
}

// create a new Private message to recipiant using secret key, pow and message to encrypt using zero or more nounce bytes
func NewEphemChatPM(recip crypto.PublicKey, sk crypto.SecretKey, pow crypto.POW, nounceExtra int, msg string) (chat EphemChat) {
	nounceSize := 32 + nounceExtra
	sender := sk.ToPublic()
	chat.Sender = sender.String()
	chat.Recip = recip.String()
	chat.Nounce = crypto.RandStr(nounceSize)
	m := make([]byte, 32+len(msg))
	copy(m[:32], chat.Nounce[0:32])
	copy(m[32:], msg[:])
	shared := crypto.KeyExchange(recip, sender, m[:32])
	chat.Message = crypto.Sym(m[32:], m[:24], &shared)
	h := crypto.HMAC(chat.Message, &shared)
	chat.Message = string(h[:]) + chat.Message
	chat.Sign(sk)
	for !chat.VerifyPOW(pow) {
		chat.Nounce = crypto.RandStr(nounceSize)
		copy(m[:32], chat.Nounce[0:32])
		shared = crypto.KeyExchange(recip, sender, m[:32])
		chat.Message = crypto.Sym(m[32:], m[:24], &shared)
		h = crypto.HMAC(chat.Message, &shared)
		copy(m[:32], h[:])
		chat.Message = string(m[:32]) + chat.Message
		chat.Sign(sk)
	}
	return
}

// mutably sign a chat
func (chat *EphemChat) Sign(sk crypto.SecretKey) (err error) {
	var data string
	data, err = bencode.EncodeString(chat.toSignForm())
	if err == nil {
		chat.Sig = sk.Sign(data)
	}
	return
}

func (chat EphemChat) toSignForm() EphemChat {
	return EphemChat{
		Sender:  chat.Sender,
		Recip:   chat.Recip,
		Target:  chat.Target,
		Nounce:  chat.Nounce,
		Message: chat.Message,
		Sig:     emptySig,
	}
}

// verify signature
func (chat EphemChat) Verify() bool {
	pk := crypto.NewPublicKey(chat.Sender)
	if pk == nil {
		return false
	}
	data, err := bencode.EncodeString(chat.toSignForm())
	if err != nil {
		return false
	}
	return pk.Verify(data, chat.Sig)
}
