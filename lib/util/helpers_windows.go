//go:build windows

package util

import (
	"net/url"
	"os/path"
	"strings"
)

func urlSchemePath(u *url.URL, scheme, path *string) {
	sch := strings.ToLower(u.Scheme)
	if len(sch) == 1 {
		// something like C:/wahtever
		sch = "file"
	}
	*outPath = u.Path
	*outSceme = sch

}
