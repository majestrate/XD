//go:build !windows

package util

import (
	"net/url"
	"strings"
)

func urlSchemePath(u *url.URL, scheme, path *string) {
	*scheme = strings.ToLower(u.Scheme)
	*path = u.Path
}
