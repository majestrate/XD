package util

import (
	"os"
	"xd/lib/log"
)

// ensure a directory is made
// returns error if it can't be made
func EnsureDir(fpath string) (err error) {
	_, err = os.Stat(fpath)
	if os.IsNotExist(err) {
		log.Debugf("create dir %s", fpath)
		err = os.Mkdir(fpath, 0700)
	}
	return
}
