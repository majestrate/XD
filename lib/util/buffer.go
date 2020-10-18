package util

import (
	"bytes"
)

type Buffer struct {
	bytes.Buffer
}

// Close implements io.Closer
func (b *Buffer) Close() error {
	return nil
}
