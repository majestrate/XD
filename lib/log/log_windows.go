//go:build windows
// +build windows

package log

var colorReset string

func (l logLevel) Color() string {
	return ""
}
