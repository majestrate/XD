package stats

import (
	"github.com/majestrate/XD/lib/util"
	"github.com/zeebo/bencode"
	"io"
)

type Tracker struct {
	history int
	rates   map[string]*util.Rate
}

func NewTracker() *Tracker {
	return &Tracker{
		history: 128,
		rates:   make(map[string]*util.Rate),
	}
}

func (t *Tracker) NewRate(name string) {
	t.rates[name] = util.NewRate(t.history)
}

func (t *Tracker) AddSample(name string, n uint64) {
	r, ok := t.rates[name]
	if ok {
		r.AddSample(n)
	}
}

func (t *Tracker) Rate(name string) (r *util.Rate) {
	r, _ = t.rates[name]
	return
}

func (t *Tracker) ForEach(v func(string, *util.Rate)) {
	for n, r := range t.rates {
		v(n, r)
	}
}

func (t *Tracker) Tick() {
	for _, r := range t.rates {
		r.Tick()
	}
}

func (t *Tracker) BEncode(w io.Writer) (err error) {
	e := bencode.NewEncoder(w)
	err = e.Encode(t.rates)
	return
}

func (t *Tracker) BDecode(r io.Reader) (err error) {
	d := bencode.NewDecoder(r)
	err = d.Decode(&t.rates)
	return
}
