package util

import (
	"net/url"
	"strconv"
)

func QueryParamString(q url.Values, name string) (string, bool) {
	v := q.Get(name)
	if v == "" {
		return "", false
	}
	return v, true
}
func QueryParamUInt64(q url.Values, name string) (uint64, bool) {
	s, ok := QueryParamString(q, name)
	if ok {
		i, err := strconv.ParseUint(s, 10, 64)
		if err == nil {
			return i, true
		}
	}
	return 0, false
}
