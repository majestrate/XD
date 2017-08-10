package util

import "fmt"

var rateUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// FormatRate formats a floating point b/s as string with closest unit
func FormatRate(rate float32) (str string) {
	var rateIdx int
	for rate > 1024.0 {
		rate /= 1024.0
		rateIdx++
	}
	str = fmt.Sprintf("%.2f%s", rate, rateUnits[rateIdx])
	return
}
