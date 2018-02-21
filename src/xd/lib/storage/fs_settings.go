package storage

import (
	"github.com/zeebo/bencode"
	"io"
)

type fsSettings struct {
	Opts map[string]string `bencode:"settings"`
}

func createSettings() fsSettings {
	return fsSettings{
		Opts: make(map[string]string),
	}
}

func (s *fsSettings) Put(key, val string) {
	s.Opts[key] = val
}

func (s *fsSettings) Get(key, fallback string) (val string) {
	var ok bool
	val, ok = s.Opts[key]
	if !ok {
		val = fallback
	}
	return
}

func (s *fsSettings) BDecode(r io.Reader) (err error) {
	dec := bencode.NewDecoder(r)
	err = dec.Decode(s)
	return
}

func (s *fsSettings) BEncode(w io.Writer) (err error) {
	enc := bencode.NewEncoder(w)
	err = enc.Encode(s)
	return
}
