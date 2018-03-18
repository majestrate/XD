package extensions

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/zeebo/bencode"
	"strings"
	"unicode"
	"xd/lib/crypto"
)

const XDEphemChat = Extension("xd_ephemchat")

// 32 byte empty sig
const emptySig = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

const XDEphemChatMetaReqMsgType = "r"
const XDEphemChatMetaRespMsgType = "R"
const XDEphemChatChatterMsgType = "C"
const XDEphemChatChatterDeliveryMsgType = "D"

const XDEphemMetaReqChannels = "c"
const XDEphemMetaReqTags = "t"

type XDFrame struct {
	Type string `bencode:"A"`
}

func XDFrameType(raw []byte) (t string) {

	if raw == nil || len(raw) < 6 {
		return
	}
	if raw[0] == 'd' && raw[1] == '1' && raw[2] == ':' && raw[3] == 'A' && raw[4] == '1' && raw[5] == ':' {
		t = string(raw[5:6])
	}
	return
}

type EphemMetaRequest struct {
	XDFrame
	Limit int    `bencode:"x"`
	Type  string `bencode:"y"`
}

type EphemMetaResponse struct {
	XDFrame
	Value []string `bencode:"w"`
	Type  string   `bencode:"y"`
}

type EphemChat struct {
	XDFrame
	Sender  string `bencode:"a"`           // sender public key
	Target  string `bencode:"b,omitempty"` // broadcast target
	Recip   string `bencode:"c,omitempty"` // pm recipiant
	Message string `bencode:"m"`           // message
	Nounce  string `bencode:"n"`           // nounce
	Version int    `bencode:"v"`           // message version (currently 0)
	Sig     string `bencode:"z"`           // signature
}

func (chat EphemChat) ToIRCLine(fallbackLine, fallbackChan string, nicklen, chanLen int, sk *crypto.SecretKey) string {
	if chat.Recip != "" {
		if sk == nil {
			return fallbackLine
		} else {
			msg, err := chat.Decrypt(*sk)
			if err == nil {
				return fmt.Sprintf(":%s PRIVMSG %s :%s", chat.SaneSender(nicklen), chat.SaneRecip(nicklen), msg)
			} else {
				fmt.Sprintf(":%s NOTICE %s :bad message: %s", chat.SaneSender(nicklen), chat.SaneRecip(nicklen), err.Error())
			}
		}
	} else if chat.Target != "" {
		return fmt.Sprintf(":%s PRIVMSG %S :%s", chat.SaneSender(nicklen), chat.SaneTarget(chanLen, fallbackChan), chat.SaneMessage())
	}
	return fallbackLine
}

func (chat EphemChat) SaneMessage() string {
	return strings.TrimFunc(chat.Message, func(r rune) bool {
		return r == '\r' || r == '\n' || !unicode.IsPrint(r)
	})
}

func (chat EphemChat) SaneTarget(l int, fallback string) string {
	if chat.Target[0] == '#' && len(chat.Target) > 1 {
		if len(chat.Target) > l {
			return chat.Target[:l]
		}
		return chat.Target
	}
	return fallback
}

func (chat EphemChat) SaneSender(l int) string {
	return hex.EncodeToString([]byte(chat.Sender))[:l]
}

func (chat EphemChat) SaneRecip(l int) string {
	return hex.EncodeToString([]byte(chat.Recip))[:l]
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
	chat.Type = XDEphemChatChatterMsgType
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
	chat.Type = XDEphemChatChatterMsgType
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
		XDFrame: XDFrame{
			Type: chat.Type,
		},
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
