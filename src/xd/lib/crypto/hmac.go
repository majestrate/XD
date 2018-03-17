package crypto

import (
	"crypto/subtle"
	"errors"
	"golang.org/x/crypto/blake2b"
	"io"
)

func HMAC(msg string, key *SymKey) (digest [32]byte) {
	x, _ := blake2b.NewXOF(32, key[:])
	io.WriteString(x, msg)
	io.ReadFull(x, digest[:])
	return
}

var ErrFailedHMAC = errors.New("failed hmac")

func VerifyHMac(msg string, digest [32]byte, key *SymKey) (err error) {
	d := HMAC(msg, key)
	if subtle.ConstantTimeCompare(d[:], digest[:]) != 1 {
		err = ErrFailedHMAC
	}
	return
}
