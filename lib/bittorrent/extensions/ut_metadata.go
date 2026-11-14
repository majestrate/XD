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

// MetadataBlock is the BEP 9 ut_metadata piece size.
const MetadataBlock = 16 * 1024

// MetadataBlockCount is the number of BEP 9 blocks in metadata of size bytes.
// A size that is an exact multiple of MetadataBlock is that many blocks, not one more.
func MetadataBlockCount(size int) uint32 {
	if size <= 0 {
		return 0
	}
	return uint32((size + MetadataBlock - 1) / MetadataBlock)
}

// MetadataPieceBounds returns the half-open byte range of one metadata block.
// ok is false when size is empty or piece is past the last block.
func MetadataPieceBounds(size int, piece uint32) (start, end int, ok bool) {
	if piece >= MetadataBlockCount(size) {
		return 0, 0, false
	}
	start = int(piece) * MetadataBlock
	end = start + MetadataBlock
	if end > size {
		end = size
	}
	return start, end, true
}

// MetadataPiece copies one BEP 9 block out of meta.
// total is the full metadata length, which BEP 9 sends as total_size.
func MetadataPiece(meta []byte, piece uint32) (data []byte, total uint32, ok bool) {
	if len(meta) > int(^uint32(0)) {
		return nil, 0, false
	}
	start, end, ok := MetadataPieceBounds(len(meta), piece)
	if !ok {
		return nil, 0, false
	}
	data = make([]byte, end-start)
	copy(data, meta[start:end])
	return data, uint32(len(meta)), true
}

// MetaData ut_metadata extension message
type MetaData struct {
	Type  int    `bencode:"msg_type"`
	Piece uint32 `bencode:"piece"`
	// Size is the full metadata length (BEP 9 total_size), not this piece's length.
	Size uint32 `bencode:"total_size"`
	Data []byte `bencode:"-"`
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
