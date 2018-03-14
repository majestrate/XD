package i2p

import "io"

func readLine(r io.Reader) (line string, err error) {
	var buff [1]byte
	for err == nil {
		_, err = r.Read(buff[:])
		if err == nil {
			line += string(buff[:])
			if buff[0] == 10 {
				break
			}
		}
	}
	return
}
