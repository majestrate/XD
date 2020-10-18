package util

import "math/rand"

func RandBoolPercent(percent uint8) bool {
	return rand.Float32()*float32(100-percent) > float32(percent)
}
