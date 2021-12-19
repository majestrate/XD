package extensions

import (
	"bytes"
	"github.com/majestrate/XD/lib/util"
	"github.com/zeebo/bencode"
)

// UTMetaData is the bittorrent extension for ut_metadata
const UTMetaData = Extension("ut_metadata")

// UTRequest msg_type for requests
const UTRequest = 0

// UTData msg_type for data
const UTData = 1

// UTReject msg_type for reject messages
const UTReject = 2

// MetaData ut_metadata extension message
type MetaData struct {
	Type  int    `bencode:"msg_type"`
	Piece uint32 `bencode:"piece"`
	Size  uint32 `bencode:"total_size"`
	Data  []byte `bencode:"-"`
}

// ParseMetadata parses a MetaData from a byteslice
func ParseMetadata(buff []byte) (md MetaData, err error) {
	r := bytes.NewReader(buff)
	d := bencode.NewDecoder(r)
	err = d.Decode(&md)
	if err == nil && md.Size > 0 {
		l := d.BytesParsed()
		md.Data = buff[l:]
	}
	return
}

// Bytes serializes a MetaData to byteslice
func (md MetaData) Bytes() []byte {
	buff := new(util.Buffer)
	if md.Type == UTData {
		bencode.NewEncoder(buff).Encode(md)
		buff.Write(md.Data)
	} else {
		bencode.NewEncoder(buff).Encode(map[string]interface{}{
			"msg_type": md.Type,
			"piece":    md.Piece,
		})
	}
	return buff.Bytes()
}
