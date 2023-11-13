//go:build !windows
// +build !windows

package log

var colorReset = "\x1b[0;0m"

func (l logLevel) Color() string {
	switch l {
	case debug:
		return "\x1b[37;0m"
	case info:
		return "\x1b[37;1m"
	case warn:
		return "\x1b[33;1m"
	default:
		return "\x1b[31;1m"
	}
}
