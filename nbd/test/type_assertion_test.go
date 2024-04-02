package test

import (
	"testing"

	"io"
	"os"
)

// Type assertion in Go is pretty fast (~20ns/3)
// so there is no reason to cache the converted value
func BenchmarkTypeAssertion(b *testing.B) {
	var object any
	object = os.Stderr
	for i := 0; i < b.N; i++ {
		var got [3]bool
		_, got[0] = object.(io.ReaderAt)
		_, got[1] = object.(io.WriterAt)
		_, got[2] = object.(error)
		if got != [...]bool{true, true, false} {
			b.Fatalf("unexpected type assertion results: %v", got)
		}
	}
}
