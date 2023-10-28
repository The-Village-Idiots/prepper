package logging

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

type checkpoint struct {
	// Index
	i int
	// Byte value
	b rune
}

// fillBuffer fills a dmesg buffer up with a given byte.
func fillBuffer(d *Dmesg, b byte) {
	fillSlice(d.buf[:], b)
}

func fillSlice(buf []byte, b byte) {
	for i := range buf {
		buf[i] = b
	}
}

func sliceMatches(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestDmesg_advance(t *testing.T) {
	tests := []struct {
		name string
		// Initial Write Pointer
		iw int
		// Advance by n
		n int
		// Expected after write pointer
		expect int
	}{
		{"Basic", 0, 5, 5},

		{"Wrap", DmesgCapacity - 1, 1, 0},
		{"Wrap2", DmesgCapacity - 5, 5, 0},

		{"Offset", DmesgCapacity - 1, 5, 4},
		{"Offset2", DmesgCapacity - 5, 10, 5},

		{"Negative", 10, -1, 9},
		{"NegativeWrap", 0, -1, DmesgCapacity - 1},
		{"NegativeWrap2", 0, -5, DmesgCapacity - 5},
	}

	d := NewDmesg()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d.head = tt.iw
			d.advance(tt.n)

			t.Logf("i=%v, n=%v, f=%v", tt.iw, tt.n, d.head)
			if d.head != tt.expect {
				t.Errorf("wrong write pointer (expect %v, got %v)", tt.expect, d.head)
			}
		})
	}
}

func TestDmesg_Write(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		iw          int
		aw          int
		checkpoints []checkpoint
	}{
		{"Basic", []byte("abcd"), 0, 4, []checkpoint{
			{0, 'a'}, {1, 'b'}, {2, 'c'}, {3, 'd'}, {4, '*'},
		}},
		{"Wrap", []byte("abcd"), DmesgCapacity - 2, 2, []checkpoint{
			{DmesgCapacity - 2, 'a'}, {DmesgCapacity - 1, 'b'}, {0, 'c'}, {1, 'd'}, {2, '*'},
		}},
		{"Full", bytes.Repeat([]byte{byte('A')}, DmesgCapacity+10), 0, 10, []checkpoint{
			{0, 'A'}, {DmesgCapacity - 1, 'A'},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDmesg()
			fillBuffer(d, byte('*'))
			d.head = tt.iw

			n, err := d.Write(tt.data)
			if n != len(tt.data) || err != nil {
				t.Error("invalid return values (expect len(buf) and nil)")
			}

			if d.head != tt.aw {
				t.Errorf("wrong write after write of len=%v, head=%v (expect %v, got %v)", len(tt.data), tt.iw, tt.aw, d.head)
			}

			for _, c := range tt.checkpoints {
				if d.buf[c.i] != byte(c.b) {
					t.Errorf("wrong byte at index %v (expect %c, got %c)", c.i, c.b, d.buf[c.i])
				}
			}
		})
	}
}

func TestDmesg_Read(t *testing.T) {
	d := NewDmesg()
	w := []byte("aaaabbbb")
	segs := [][]byte{[]byte("aaaa"), []byte("bbbb")}
	d.Write(w)

	r := d.Reader()
	buf := make([]byte, 4)

	i, cu := 0, 0
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err != io.EOF {
				t.Errorf("unexpected error (err: %v)", err)
			}

			break
		}

		cu += n

		if i >= len(segs) {
			t.Errorf("unexpected extra read segment (already read %v, got another?)", i)
		}

		if i < len(segs) && !sliceMatches(buf, segs[i]) {
			t.Errorf("invalid segment (expect: %v, got: %v)", segs[i], buf)
		}
		i++
	}

	if cu < len(w) {
		t.Errorf("EOF returned before all data read (bytes read: %v)", cu)
	}
	if cu > len(w) {
		t.Errorf("extra data read (buf: %v, overread of %v bytes)", buf, cu-len(w))
	}
}

func TestDmesg_ReadWrap(t *testing.T) {
	d := NewDmesg()
	lower, upper := bytes.Repeat([]byte("A"), DmesgCapacity/2), bytes.Repeat([]byte("B"), DmesgCapacity/2)
	res := append(lower, upper...)

	// Make an artificial wrapping scenario
	d.head = DmesgCapacity / 2
	n, err := d.Write(res)
	if err != nil || n != len(res) {
		t.Fatalf("unexpected failed write (err: %v, n: %v)", err, n)
	}

	r := d.Reader()
	buf := make([]byte, 4)

	cu := 0
	for i := 0; i < 2048; i++ {
		n, err := r.Read(buf)
		cu += n
		if err != nil {
			if err != io.EOF {
				t.Errorf("unexpected error (err: %v)", err)
			}

			break
		}

		// Should be in order, so lower half is entirely A (despite
		// actual buffer layout being reverse).
		match := []byte("AAAA")
		if cu > DmesgCapacity/2 {
			match = []byte("BBBB")
		}

		if !reflect.DeepEqual(match, buf) {
			t.Errorf("bad read at byte %v (expect: %v, read: %v)", cu-n, match, buf)
		}
	}

	if cu != len(res) {
		t.Errorf("unexpected EOF at byte %v (%v left to read)", cu, len(res)-cu)
	}
}

func TestDmesg_ReadAllWrap(t *testing.T) {
	d := NewDmesg()
	lower, upper := bytes.Repeat([]byte("A"), DmesgCapacity/2), bytes.Repeat([]byte("B"), DmesgCapacity/2)
	res := append(lower, upper...)

	// Make an artificial wrapping scenario
	d.head = DmesgCapacity / 2
	d.Write(res)

	r := d.Reader()
	out, err := io.ReadAll(r)

	if err != nil {
		t.Fatalf("unexpected error (err: %v)", err)
	}
	if len(out) != len(res) {
		t.Errorf("bad length (expect: %v, got: %v)", len(res), len(out))
	}
}

func TestDmesg_ReadAll(t *testing.T) {
	buf := bytes.Repeat([]byte{byte('A')}, 100)
	d := NewDmesg()
	d.Write(buf)

	out, err := io.ReadAll(d.Reader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != len(buf) {
		t.Errorf("wrong buffer output length (expect: %v, got: %v)", len(buf), len(out))
	}
}
