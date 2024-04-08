package test

import (
	"testing"
)

// Creating channels in Go is rather fast (65ns, 1 alloc, 96B),
// it's okay to do so once per cache block (10-1000 times per second)
func BenchmarkChanCreateClose(b *testing.B) {
	var ch chan struct{}
	for i := 0; i < b.N; i++ {
		ch = make(chan struct{})
		close(ch)
	}
}
