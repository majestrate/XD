package crypto

import (
	"crypto/sha512"
	"edwards25519"
	"golang.org/x/crypto/curve25519"
)

func seedToKeyPair(seed []byte) (pk, sk []byte) {

	h := sha512.Sum512(seed[0:32])
	sk = h[:]
	sk[0] &= 248
	sk[31] &= 127
	sk[31] |= 64
	// scalarmult magick shit
	pk = scalarBaseMult(sk[0:32])
	copy(sk[0:32], seed[0:32])
	copy(sk[32:64], pk[0:32])
	return
}

func scalarMult(shared, a, b *[32]byte) {
	curve25519.ScalarMult(shared, a, b)
}

func scalarBaseMult(sk []byte) (pk []byte) {
	var skey [32]byte
	var pkey [32]byte
	copy(skey[:], sk[0:32])
	var h edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&h, &skey)
	h.ToBytes(&pkey)
	pk = pkey[:]
	return
}
