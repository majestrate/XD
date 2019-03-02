package fs

import (
	"io"
	"os"
)

type ReadFile interface {
	io.ReadCloser
	io.ReaderAt
}

type WriteFile interface {
	io.WriteCloser
	io.WriterAt
	Sync() error
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
	// remove single file
	Remove(fpath string) error
	// Remove all in filepath
	RemoveAll(fpath string) error
	// Join path
	Join(parts ...string) string
	// move file
	Move(oldPath, newPath string) error
	// split path into dirname, basename
	Split(path string) (string, string)
	// call stat()
	Stat(path string) (os.FileInfo, error)
}
