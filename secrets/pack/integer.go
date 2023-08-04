package pack

import (
	"unicode/utf8"
)

// Piggyback on UTF-8 variable width encoding for storing non-negative integers
type Uint rune

func (i Uint) Bytes() []byte {
	var r = rune(i)
	if r > utf8.MaxRune {
		panic("UTF-8 codepoint overflow")
	}
	if r < 0 {
		panic("UTF-8 negative codepoint")
	}
	b := make([]byte, utf8.UTFMax)
	n := utf8.EncodeRune(b, rune(i))
	return b[:n]
}
