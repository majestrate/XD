package util

type discard struct {
}

func (d discard) Write(data []byte) (n int, err error) {
	n = len(data)
	return
}

func (d discard) Close() (err error) {
	return
}

var Discard discard
