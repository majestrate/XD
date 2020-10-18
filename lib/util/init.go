package util

import (
	"time"
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

func StartedAt() time.Time {
	return startTime
}
