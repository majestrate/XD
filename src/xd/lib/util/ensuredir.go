package util

import (
	"os"
)

// ensure a directory is made
// returns error if it can't be made
func EnsureDir(fpath string) (err error) {
	_, err = os.Stat(fpath)
	if os.IsNotExist(err) {
		err = os.Mkdir(fpath, 0700)
	}
	return
}
