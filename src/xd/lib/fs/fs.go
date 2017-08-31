package fs

import (
	"io"
)

type ReadFile interface {
	io.ReadCloser
	io.ReaderAt
}

type WriteFile interface {
	io.WriteCloser
	io.WriterAt
}

type Driver interface {
	io.Closer
	// open any underlying contexts
	Open() error
	// open file ready only
	OpenFileReadOnly(fpath string) (ReadFile, error)
	// open file write only
	OpenFileWriteOnly(fpath string) (WriteFile, error)
	// return true if file exists
	FileExists(fpath string) bool
	// ensure a directory exists
	EnsureDir(fpath string) error
	// ensire a file exists and is of size sz
	EnsureFile(fpath string, sz uint64) error
	// filepath.Glob lookalike
	Glob(str string) ([]string, error)
}
