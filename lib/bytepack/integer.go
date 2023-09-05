package bytepack

import (
	"errors"
	"unicode/utf8"
)

var (
	ErrOverflowUTF8 = errors.New("UTF-8 codepoint overflow")
	ErrNegativeUTF8 = errors.New("UTF-8 negative codepoint")
)

// Piggyback on UTF-8 variable width encoding
// for storing relatively small unsigned integers
type Uint rune

func (i Uint) Bytes() ([]byte, error) {
	r := rune(i)
	if r > utf8.MaxRune {
		return nil, ErrOverflowUTF8
	}
	if r < 0 {
		return nil, ErrNegativeUTF8
	}
	var b [utf8.MaxRune]byte
	n := utf8.EncodeRune(b[:], r)
	return b[:n], nil
}
