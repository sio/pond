package bytepack

import (
	"unicode/utf8"
)

// Piggyback on UTF-8 variable width encoding
// for storing relatively small unsigned integers
type Uint rune

// Represent integer in binary format.
// Allocates a new byte slice for the result
func (i Uint) Bytes() []byte {
	var b = make([]byte, utf8.UTFMax)
	n := i.Encode(b)
	return b[:n]
}

// Write integer bytes into the given slice.
// Return number of bytes written
func (i Uint) Encode(to []byte) int {
	r := rune(i)
	if r > utf8.MaxRune {
		r = utf8.RuneError
	}
	if r < 0 {
		r = utf8.RuneError
	}
	return utf8.EncodeRune(to, r)
}

// Return size of binary representation (in bytes)
func (i Uint) Size() int {
	var x [utf8.UTFMax]byte
	return i.Encode(x[:])
}
