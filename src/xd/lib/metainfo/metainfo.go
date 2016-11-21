package metainfo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"xd/lib/bencode"
)


type FileInfo struct {
	// length of file
	Length int64
	// relative path of file
	Path string
}

// implements bencode.Serializable
func (i FileInfo) BEncode(w io.Writer) (err error) {
	fmt.Fprintf(w, "d") // begin dict
	fmt.Fprintf(w, "6:lengthi%de", i.Length) // length = i.Length
	fmt.Fprintf(w, "4:pathl") // begin path list
	for _, f := range strings.Split(i.Path, string(filepath.Separator)) {
		fmt.Fprintf(w, "%d:%s", len(f), f) // path entry
	}
	fmt.Fprintf(w, "e") // end path list
	_, err = fmt.Fprintf(w, "e") // end dict
	return
}

// implements bencode.Serializable
func (i FileInfo) BDecode(r io.Reader) (err error) {
	var buff [9]byte
	_, err = r.Read(buff[:])
	if buff[0] != 100 {
		err = errors.New("expected dict by got "+string(buff[:]))
		return
	}
	if ! bytes.Equal(buff[1:], []byte("6:length")) {
		err = errors.New("expected length but got "+string(buff[1:]))
		return
	}
	i.Length, err = bencode.ReadInt(r)
	if err == nil {
		_, err = r.Read(buff[2:])
		if ! bytes.Equal(buff[2:], []byte("4:pathl")) {
			err = errors.New("expected path but got "+string(buff[2:]))
			return
		}
		var l [][]byte
		l, err = bencode.ReadStringList(r)
		for _, f := range l {
			i.Path = filepath.Join(i.Path, string(f))
		}
		// read 'e'
		_, err = r.Read(buff[8:])
	}
	return
}

// info section of torrent file
type Info struct {
	// length of pices in bytes
	PieceLength int64
	// piece data
	Pieces []byte
	// name of root file
	Path string
	// file metadata
	Files []FileInfo
	// private torrent
	Private int64
}

func (i Info) BEncode(w io.Writer) (err error) {
	if len(i.Files) == 0 {
		err = errors.New("no files in info section")
		return
	}
	fmt.Fprintf(w, "d") // begin dict
	
	if len(i.Files) > 1 {
		// multifile mode
		fmt.Fprintf(w, "d4:files") // begin files dict
		for _, f := range i.Files {
			err = f.BEncode(w)
			if err != nil {
				return 
			}
		}
		fmt.Fprintf(w, "e") // end files dict
	} else {
		// single file mode
		// length = i.Files[0].length
		fmt.Fprintf(w, "6:length")
		err = bencode.WriteInt(w, i.Files[0].Length)
		if err != nil {
			return
		}
	}
	// name = i.Path
	fmt.Fprintf(w, "4:name")
	err = bencode.WriteString(w, i.Path)
	if err != nil {
		return
	}
	// piece length 
	err = bencode.WriteString(w, "piece length")
	if err != nil {
		return
	}
	err = bencode.WriteInt(w, i.PieceLength)
	if err != nil {
		return
	}
	// actual piece data
	err = bencode.WriteString(w, "pieces")
	if err != nil {
		return
	}
	// piece data length
	l := len(i.Pieces)
	_, err = fmt.Fprintf(w, "%d:", l)
	if err != nil {
		return
	}
	// actual piece data
	var n, offset int
	for err == nil && offset < l {
		n, err = w.Write(i.Pieces[offset:])
		if err == nil {
			offset += n
		}
	}
	// private flag
	err = bencode.WriteInt(w, i.Private)
	if err == nil {
		_, err = fmt.Fprintf(w, "e"); // end dict
	}
	return
}


type TorrentFile struct {
	Info Info
	Announce string
	AnnounceList []string
	Created int64
	Comment string
	CreatedBy string
	Encoding string
}


// implements bencode.Serializable
func (t *TorrentFile) BDecode(r io.Reader) (err error) {
	return
}

// implements bencode.Serializable
func (t *TorrentFile) BEncode(w io.Writer) (err error) {
	return
}
