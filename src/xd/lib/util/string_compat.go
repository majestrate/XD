// +build !go1.8

package util

import "bytes"

func StringCompare(a, b string) int {
	return bytes.Compare([]byte(a), []byte(b))
}
