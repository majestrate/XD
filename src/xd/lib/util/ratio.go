package util

import "math"

func Ratio(tx, rx float64) (r float64) {
	if rx > 0 {
		r = tx / rx
	} else if tx > 0 {
		r = math.Inf(1)
	}
	return
}
