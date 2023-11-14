//go:build windows

package util

import (
	"net/url"
	"strings"
)

func urlSchemePath(u *url.URL, scheme, path *string) {
	sch := strings.ToLower(u.Scheme)
	if len(sch) == 1 {
		// something like C:/wahtever
		sch = "file"
	}
	*path = u.String()
	*scheme = sch
}
