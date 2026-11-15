package bittorrent

import "testing"

func TestBitfieldWireLength(t *testing.T) {
	cases := []struct {
		bits uint32
		want int
	}{
		{0, 0},
		{1, 1},
		{8, 1},
		{9, 2},
		{2048, 256},
		{2049, 257},
	}
	for _, tc := range cases {
		bf := NewBitfield(tc.bits, nil)
		if got := len(bf.Data); got != tc.want {
			t.Errorf("alloc bits=%d len=%d, want %d", tc.bits, got, tc.want)
		}
		msg := bf.ToWireMessage()
		n := int(msg[0])<<24 | int(msg[1])<<16 | int(msg[2])<<8 | int(msg[3])
		if n != tc.want+1 {
			t.Errorf("wire bits=%d length prefix %d, want %d", tc.bits, n, tc.want+1)
		}
		if len(msg) != 5+tc.want {
			t.Errorf("wire bits=%d message len %d, want %d", tc.bits, len(msg), 5+tc.want)
		}
	}
}

func TestBitfieldWireTruncatesStoredPadding(t *testing.T) {
	stored := make([]byte, 257)
	stored[255] = 0xff
	stored[256] = 0xff
	bf := NewBitfield(2048, stored)
	msg := bf.ToWireMessage()
	payload := msg[5:]
	if len(payload) != 256 {
		t.Fatalf("payload len %d", len(payload))
	}
	if payload[255] != 0xff {
		t.Fatalf("last used byte %#x", payload[255])
	}
}

func TestBitfieldSpareBitsAreZero(t *testing.T) {
	bf := NewBitfield(9, []byte{0xff, 0xff})
	payload := bf.ToWireMessage()[5:]
	if len(payload) != 2 || payload[0] != 0xff || payload[1] != 0x80 {
		t.Fatalf("payload %#x", payload)
	}
}
