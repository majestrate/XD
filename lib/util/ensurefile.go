package util

import (
	"github.com/majestrate/XD/lib/log"
	"io"
	"os"
	"path/filepath"
)

// ensure a file and its parent directory exists
func EnsureFile(fpath string, size uint64) (err error) {
	d, _ := filepath.Split(fpath)
	if d != "" {
		err = EnsureDir(d)
	}
	if err == nil {
		_, err = os.Stat(fpath)
		if os.IsNotExist(err) {
			log.Debugf("create file %s", fpath)
			var f *os.File
			f, err = os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0666)
			if err == nil {
				// fill with zeros
				if size > 0 {
					_, err = io.CopyN(f, Zero, int64(size))
				}
				f.Close()
			}
		}
	}
	return
}
