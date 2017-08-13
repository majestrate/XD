package util

import (
	"time"
)

type Rate struct {
	samples       [10]uint64
	lastSampleIdx int
	lastTick      int64
}

func (r *Rate) Tick() {
	r.lastSampleIdx = (r.lastSampleIdx + 1) % len(r.samples)
	r.lastTick = time.Now().Unix()
	r.samples[r.lastSampleIdx] = 0
}

func (r *Rate) AddSample(n uint64) {
	r.samples[r.lastSampleIdx] += n
}

func (r *Rate) Rate() float64 {
	mean := uint64(0)
	for idx := range r.samples {
		mean += r.samples[idx]
	}
	mean /= uint64(len(r.samples))
	now := float64(time.Now().Unix() - r.lastTick)
	if now <= 0 {
		now = 1.0
	}
	return float64(mean) / now
}
