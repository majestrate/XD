package crypto

import (
	"crypto/rand"
	"io"
)

func RandStr(l int) string {
	m := make([]byte, l)
	io.ReadFull(rand.Reader, m)
	return string(m)
}
