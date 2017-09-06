// +build windows

package log

const colorReset = ""

func (l logLevel) Color() string {
	return ""
}
