package i2p

import "io"

func readLine(r io.Reader, buff []byte) (line string, err error) {
	var n int
	for err == nil {
		n, err = r.Read(buff[:])
		if err == nil {
			line += string(buff[:])
			if buff[n-1] == 10 {
				break
			}
		}
	}
	return
}
