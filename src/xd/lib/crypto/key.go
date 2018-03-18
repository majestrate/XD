package crypto

import (
	"crypto/sha256"
	"golang.org/x/crypto/ed25519"
)

type SecretKey [64]byte

type PublicKey [32]byte

type SymKey [32]byte

func (k SecretKey) String() string {
	return string(k[:])
}

func (k *SecretKey) Bytes() []byte {
	return (*k)[:]
}

func (k PublicKey) Bytes() []byte {
	return k[:]
}

func KeyGen() *SecretKey {
	seed := RandStr(32)
	_, sk := seedToKeyPair([]byte(seed))
	var k SecretKey
	copy(k[:], sk[:])
	return &k
}

func (k SecretKey) Sign(msg string) (sig string) {
	s := ed25519.Sign(ed25519.PrivateKey(k[:]), []byte(msg))
	sig = string(s)
	return
}

func (k PublicKey) Verify(msg, sig string) bool {
	return ed25519.Verify(ed25519.PublicKey(k[:]), []byte(msg), []byte(sig))
}

func (k PublicKey) String() string {
	return string(k[:])
}

func (sk *SecretKey) ToPublic() (pk PublicKey) {
	copy(pk[:], sk.Bytes()[32:])
	return
}

func (pk *PublicKey) toBytes() *[32]byte {
	var k [32]byte
	copy(k[:], pk[:])
	return &k
}

func NewPublicKey(k string) (pk *PublicKey) {
	if len(k) == 32 {
		pk = new(PublicKey)
		copy(pk[:], k[:])
	}
	return
}

func KeyExchange(recip, sender PublicKey, nounce []byte) (sharedKey SymKey) {
	var shared [32]byte
	scalarMult(&shared, recip.toBytes(), sender.toBytes())
	var m [32 * 3]byte
	copy(m[0:32], shared[:])
	copy(m[32:64], recip[:])
	copy(m[64:], sender[:])
	var b []byte
	b = append(b, nounce...)
	b = append(b, m[:]...)
	h := sha256.Sum256(b)
	copy(sharedKey[:], h[:])
	return
}
