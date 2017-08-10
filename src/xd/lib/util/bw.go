package util

import (
	"fmt"
	"math"
)

var rateUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// FormatRate formats a floating point b/s as string with closest unit
func FormatRate(rate float64) (str string) {
	if math.IsInf(rate, 0) {
		str = "infinity"
		return
	}
	var rateIdx int
	for rate > 1024.0 {
		rate /= 1024.0
		rateIdx++
	}
	str = fmt.Sprintf("%.2f%s/sec", rate, rateUnits[rateIdx])
	return
}
