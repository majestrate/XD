package util

import "net/url"

func SchemePath(u *url.URL) (scheme string, path string) {
	urlSchemePath(u, &scheme, &path)
	return
}
