package util

import (
	"time"
)

type Rate struct {
	samples  []uint64
	lastTick int64
}

func (r *Rate) Tick() {
	r.lastTick = time.Now().Unix()
}

func (r *Rate) AddSample(n uint64) {
	r.samples = append(r.samples, n)
}

func (r *Rate) ClearSamples() {
	r.samples = []uint64{}
}

func (r *Rate) Rate() float64 {
	mean := uint64(0)
	for idx := range r.samples {
		mean += r.samples[idx]
	}
	mean /= uint64(len(r.samples))
	now := float64(time.Now().Unix() - r.lastTick)
	return float64(mean) / now
}
