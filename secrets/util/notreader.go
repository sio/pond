package util

import "errors"

// FailingReader implements io.Reader but all Read() calls fail with an error.
//
// This is useful as a replacement for crypto/rand.Reader when only
// deterministic signatures are required.
var FailingReader = &failingReader{}

type failingReader struct{}

func (n *failingReader) Read(p []byte) (int, error) {
	return 0, errors.New("random numbers make signature non-deterministic")
}
