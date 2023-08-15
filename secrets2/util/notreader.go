package util

import "errors"

// AntiReader implements io.Reader but all Read() calls fail with an error.
//
// This is useful as a replacement for crypto/rand.Reader when only
// deterministic signatures are required.
var (
	AntiReader    = &antiReader{}
	ErrAntiReader = errors.New("reading not allowed")
)

type antiReader struct{}

func (n *antiReader) Read(p []byte) (int, error) {
	return 0, ErrAntiReader
}
