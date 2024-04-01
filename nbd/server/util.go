package server

import (
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
		if done != end && err != nil {
			return err
		}
		n -= done
	}
	return nil
}

// We do not care about data that gets written here.
// Data races and spontaneous overwriting does not bother us.
// Never read data from this buffer!
var discardBuf [4096]byte
