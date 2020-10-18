package util

import (
	"crypto/rand"
	"encoding/base32"
	"io"
)

func RandStr(l int) string {
	buff := make([]byte, l)
	io.ReadFull(rand.Reader, buff)
	return base32.StdEncoding.EncodeToString(buff)[:l]
}
