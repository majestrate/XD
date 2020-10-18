package util

import (
	"os"
)

// CheckFile returns true if a file exists
func CheckFile(fpath string) (exists bool) {
	_, err := os.Stat(fpath)
	exists = !os.IsNotExist(err)
	return
}
