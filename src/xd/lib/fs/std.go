package fs

import (
	"os"
	"path/filepath"
	"xd/lib/util"
)

type stdFs struct{}

var STD stdFs

func (f stdFs) Open() error {
	return nil
}

func (f stdFs) Close() error {
	return nil
}

func (f stdFs) EnsureDir(fname string) error {
	return util.EnsureDir(fname)
}

func (f stdFs) EnsureFile(fname string, sz uint64) error {
	return util.EnsureFile(fname, sz)
}

func (f stdFs) FileExists(fname string) bool {
	return util.CheckFile(fname)
}

func (f stdFs) Glob(glob string) ([]string, error) {
	return filepath.Glob(glob)
}

func (f stdFs) OpenFileReadOnly(fname string) (ReadFile, error) {
	return os.Open(fname)
}

func (f stdFs) OpenFileWriteOnly(fname string) (WriteFile, error) {
	return os.OpenFile(fname, os.O_CREATE|os.O_WRONLY, 0600)
}
