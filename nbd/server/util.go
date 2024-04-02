package server

import (
	"fmt"
	"io"
)

// Discard specified number of bytes from io.Reader
func discard(r io.Reader, n int) error {
	for n > 0 {
		end := n
		if end > cap(discardBuf) {
			end = cap(discardBuf)
		}
		done, err := r.Read(discardBuf[:end])
		n -= done
		if err != nil && n != 0 {
			return err
		}
	}
	return nil
}

// We do not care about data that gets written here.
// Data races and spontaneous overwriting does not bother us.
// Never read data from this buffer!
var discardBuf [4096]byte

// Simple io.Writer interface for a byte slice.
// Does not capture and mangle the slice like bytes.Buffer does.
type byteWriter struct {
	buf []byte
}

func (w *byteWriter) Write(p []byte) (n int, err error) {
	n = copy(w.buf[len(w.buf):cap(w.buf)], p)
	w.buf = w.buf[:len(w.buf)+n]
	if n != len(p) {
		return n, fmt.Errorf("buffer full")
	}
	return n, nil
}

func (w *byteWriter) Bytes() []byte {
	return w.buf
}
