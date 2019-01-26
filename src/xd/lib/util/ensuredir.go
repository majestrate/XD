package util

import (
	"os"
)

// ensure a directory is made
// returns error if it can't be made
func EnsureDir(fpath string) (err error) {
	err = os.MkdirAll(fpath, os.ModePerm)
	return
}
