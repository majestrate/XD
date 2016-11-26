package util

import (
	"os"
	"path/filepath"
)

// ensure a directory is made
// returns error if it can't be made
func EnsureDir(fpath string) (err error) {
	d, _ := filepath.Split(fpath)
	if len(d) > 0 {
		err = EnsureDir(d)
	}
	_, err = os.Stat(fpath)
	if os.IsNotExist(err) {
		err = os.Mkdir(fpath, 0700)
	}
	return
}
