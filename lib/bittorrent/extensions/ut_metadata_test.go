package extensions

import (
	"bytes"
	"testing"
)

func patterned(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func TestMetadataBlockCount(t *testing.T) {
	cases := []struct {
		size int
		want uint32
	}{
		{0, 0},
		{1, 1},
		{1540, 1},
		{MetadataBlock, 1},
		{MetadataBlock + 1, 2},
		{28114, 2},
		{2 * MetadataBlock, 2},
		{154488, 10},
	}
	for _, tc := range cases {
		if got := MetadataBlockCount(tc.size); got != tc.want {
			t.Errorf("MetadataBlockCount(%d) = %d, want %d", tc.size, got, tc.want)
		}
	}
}

func TestMetadataPieceBEP9(t *testing.T) {
	// Sizes from https://github.com/majestrate/XD/issues/132.
	// Piece 0 must be the first 16 KiB (or the whole file when smaller),
	// never a reject and never the entire multi-block file.
	cases := []struct {
		n     int
		piece uint32
		want  int
		ok    bool
	}{
		{1540, 0, 1540, true},
		{1540, 1, 0, false},
		{28114, 0, MetadataBlock, true},
		{28114, 1, 28114 - MetadataBlock, true},
		{28114, 2, 0, false},
		{154488, 0, MetadataBlock, true},
		{154488, 9, 154488 - 9*MetadataBlock, true},
		{154488, 10, 0, false},
		{MetadataBlock, 0, MetadataBlock, true},
		{MetadataBlock, 1, 0, false},
		{2 * MetadataBlock, 0, MetadataBlock, true},
		{2 * MetadataBlock, 1, MetadataBlock, true},
		{2 * MetadataBlock, 2, 0, false},
		{0, 0, 0, false},
	}
	for _, tc := range cases {
		meta := patterned(tc.n)
		data, total, ok := MetadataPiece(meta, tc.piece)
		if ok != tc.ok {
			t.Errorf("n=%d piece=%d ok=%v, want %v", tc.n, tc.piece, ok, tc.ok)
			continue
		}
		if !ok {
			continue
		}
		if total != uint32(tc.n) {
			t.Errorf("n=%d piece=%d total_size=%d, want %d", tc.n, tc.piece, total, tc.n)
		}
		if len(data) != tc.want {
			t.Errorf("n=%d piece=%d len=%d, want %d", tc.n, tc.piece, len(data), tc.want)
		}
		start := int(tc.piece) * MetadataBlock
		if !bytes.Equal(data, meta[start:start+len(data)]) {
			t.Errorf("n=%d piece=%d bytes are not that block", tc.n, tc.piece)
		}
		if tc.n > MetadataBlock && tc.piece == 0 && len(data) == tc.n {
			t.Errorf("n=%d piece 0 returned the entire metadata", tc.n)
		}

		md := MetaData{Type: UTData, Piece: tc.piece, Size: total, Data: data}
		raw := md.Bytes()
		prefix := []byte("d8:msg_typei1e5:piecei")
		if !bytes.HasPrefix(raw, prefix) {
			t.Fatalf("n=%d piece=%d wire prefix %q", tc.n, tc.piece, raw)
		}
		parsed, err := ParseMetadata(raw)
		if err != nil {
			t.Fatalf("n=%d piece=%d parse: %v", tc.n, tc.piece, err)
		}
		if parsed.Type != UTData || parsed.Piece != tc.piece || parsed.Size != total {
			t.Fatalf("n=%d piece=%d parsed %+v", tc.n, tc.piece, parsed)
		}
		if !bytes.Equal(parsed.Data, data) {
			t.Fatalf("n=%d piece=%d parsed data len %d", tc.n, tc.piece, len(parsed.Data))
		}
		if len(data) < tc.n && len(raw) >= tc.n {
			t.Fatalf("n=%d piece=%d wire length %d carries more than the requested block", tc.n, tc.piece, len(raw))
		}
	}
}

func TestMetadataRequestAndRejectWire(t *testing.T) {
	req := MetaData{Type: UTRequest, Piece: 0}.Bytes()
	if !bytes.Equal(req, []byte("d8:msg_typei0e5:piecei0ee")) {
		t.Fatalf("request wire %q", req)
	}
	rej := MetaData{Type: UTReject, Piece: 0}.Bytes()
	if !bytes.Equal(rej, []byte("d8:msg_typei2e5:piecei0ee")) {
		t.Fatalf("reject wire %q", rej)
	}
}
