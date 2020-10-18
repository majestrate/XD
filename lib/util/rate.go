package util

import (
	"github.com/zeebo/bencode"
	"io"
	"time"
)

// (magnitude, time)
type RateSample [2]uint64

func (s RateSample) Value() uint64 {
	return s[0]
}

func (s RateSample) Time() time.Time {
	return time.Unix(int64(s[1]), 0)
}

func (s *RateSample) Clear() {
	s.Set(0)
}

func (s *RateSample) Set(n uint64) {
	(*s)[0] = n
	(*s)[1] = uint64(time.Now().Unix())
}

func (s *RateSample) Add(n uint64) {
	(*s)[0] += n
}

type Rate struct {
	Samples       []RateSample
	lastSampleIdx int
}

func NewRate(sampleLen int) *Rate {
	return &Rate{
		Samples: make([]RateSample, sampleLen),
	}
}

func (r *Rate) BEncode(w io.Writer) (err error) {
	e := bencode.NewEncoder(w)
	err = e.Encode(r)
	return
}

func (r *Rate) BDecode(rd io.Reader) (err error) {
	d := bencode.NewDecoder(rd)
	err = d.Decode(r)
	return
}

func (r *Rate) Tick() {
	r.lastSampleIdx = (r.lastSampleIdx + 1) % len(r.Samples)
	r.Samples[r.lastSampleIdx].Clear()
}

func (r *Rate) AddSample(n uint64) {
	r.Samples[r.lastSampleIdx].Add(n)
}

func (r *Rate) Max() (max uint64) {
	for idx := range r.Samples {
		val := r.Samples[idx].Value()
		if val > max {
			max = val
		}
	}
	return
}

func (r *Rate) Current() (cur uint64) {
	cur = r.Samples[r.lastSampleIdx].Value()
	return
}

func (r *Rate) Min() (min uint64) {
	min = ^uint64(0)
	for idx := range r.Samples {
		val := r.Samples[idx].Value()
		if val < min {
			min = val
		}
	}
	return
}

func (r *Rate) PrevTickTime() time.Time {
	if r.lastSampleIdx == 0 {
		return r.Samples[len(r.Samples)-1].Time()
	}
	return r.Samples[r.lastSampleIdx-1].Time()
}

func (r *Rate) Mean() float64 {
	lastTick := r.PrevTickTime().Unix()
	sum := uint64(0)
	for idx := range r.Samples {
		sum += r.Samples[idx].Value()
	}
	sum /= uint64(len(r.Samples))
	now := float64(time.Now().Unix() - lastTick)
	if now <= 0 {
		now = 1.0
	}
	return float64(sum) / now
}
