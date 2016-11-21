package bencode

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

// interface to implement if a type is bencode-able
type Serializable interface {
	// bencode to writer
	BEncode(w io.Writer) error
	// bdecode from reader
	BDecode(r io.Reader) error
}

// write a string literal
func WriteString(w io.Writer, str string) (err error) {
	_, err = fmt.Fprintf(w, "%d:%s", len(str), str)
	return
}

// write a string read from a Reader given length
func WriteStringFrom(w io.Writer, r io.Reader, l int64) (err error) {
	_, err = fmt.Fprintf(w, "%d:", l)
	if err == nil {
		_, err = io.CopyN(w, r, l)
	}
	return
}

// read a list of strings
func ReadStringList(r io.Reader) (l [][]byte, err error) {
	var buff [1]byte
	_, err = r.Read(buff[:])
	if err == nil {
		if buff[0] == 108 {
			var str []byte
			var lstr []byte
			var slen int
			for err == nil {
				_, err = r.Read(buff[:])
				if err == nil {
					if buff[0] == 58 {
						// hit delimiter
						slen, err = strconv.Atoi(string(lstr))
						// read string contents
						for err == nil && slen > 0 {
							_, err = r.Read(buff[:])
							slen --
							str = append(str, buff[0])
						}
						if err == nil {
							l = append(l, str)
							str = nil
							lstr = nil
						} else {
							break
						}
					} else if buff[0] == 101 {
						// end of list
						break
					} else {
						lstr = append(lstr, buff[0])
					}
				}
			}
		} else {
			err = errors.New("expected list but got "+string(buff[:]))
		}
	}
	return
}

// read string from reader into writer
func ReadString(r io.Reader, w io.Writer) (err error) {
	var buff [1]byte
	var lbuff []byte
	var l int64
	for err == nil {
		_, err = r.Read(buff[:])
		if err == nil {
			if buff[0] == 58 {
				// delimiter hit
				break
			} else {
				lbuff = append(lbuff, buff[0])
			}
		}
	}
	l, err = strconv.ParseInt(string(lbuff), 10, 64)
	if err == nil {
		_, err = io.CopyN(w, r, l)
	}
	return
}

// write an int
func WriteInt(w io.Writer, i int64) (err error) {
	_, err = fmt.Fprintf(w, "i%de", i)
	return
}

// read an int
func ReadInt(r io.Reader) (i int64, err error) {
	var buff [1]byte
	_, err = r.Read(buff[:])
	if err == nil {
		if buff[0] == 105 {
			var ibuff []byte
			for err == nil {
				_, err = r.Read(buff[:])
				if buff[0] == 101 {
					// end
					i, err = strconv.ParseInt(string(ibuff), 10, 64)
					break
				} else {
					ibuff = append(ibuff, buff[0])
				}
			}
		}
	}
	return
}
