package crypto

import (
	"golang.org/x/crypto/salsa20"
)

func Sym(msg []byte, n []byte, k *SymKey) string {
	out := make([]byte, len(msg))
	var sk [32]byte
	copy(sk[:], k[:])
	salsa20.XORKeyStream(out, msg, n, &sk)
	return string(out)
}
