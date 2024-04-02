package server

import (
	"testing"

	"fmt"
)

func TestDiscard(t *testing.T) {
	size := len(discardBuf)
	for _, n := range []int{4, 8, 15, 16, 23, 42, 1000, size - 1, size, size + 1, size * 2, size*3 + 1, (size << 3) + 1} {
		r := new(trackingReader)
		err := discard(r, n)
		if err != nil {
			t.Fatalf("read error: %v", err)
		}
		if r.Count() != n {
			t.Errorf("read count mismatch: want %d, got %d", n, r.Count())
		}
	}
}

// Track how may byte were read from this reader
type trackingReader int

func (r trackingReader) String() string {
	return fmt.Sprintf("{%d bytes read}", r)
}

func (r trackingReader) Count() int {
	return int(r)
}

func (r *trackingReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(i)
	}
	*r += trackingReader(len(p))
	return len(p), nil
}
