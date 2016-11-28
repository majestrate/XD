package util

import (
	"path/filepath"
	"os"
	"io"
)

// ensure a file and its parent directory exists
func EnsureFile(fpath string, size int64) (err error) {
	d, _ := filepath.Split(fpath)
	err = EnsureDir(d)
	if err == nil {
		_, err = os.Stat(fpath)
		if os.IsNotExist(err) {
			var f *os.File
			f, err = os.OpenFile(fpath, os.O_CREATE | os.O_WRONLY, 0600)
			if err == nil {
				// fill with zeros
			_, err = io.CopyN(f, Zero, size)
				f.Close()
			}
		}
	}
	return
}
