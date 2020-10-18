package util

type zeroReader struct {
}

func (z *zeroReader) Read(d []byte) (int, error) {
	i := 0
	for i < len(d) {
		d[i] = 0
		i++
	}
	return len(d), nil
}

// reader that reads zeros
var Zero = new(zeroReader)
