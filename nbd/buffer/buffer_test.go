package buffer

import (
	"testing"
)

func BenchmarkBuffer(b *testing.B) {
	var buf []byte
	var warmup [100][]byte
	for i := range warmup {
		warmup[i] = buffer.Get()
	}
	for i := range warmup {
		buffer.Put(warmup[i])
	}
	for i := 0; i < b.N; i++ {
		buf = buffer.Get()
		buffer.Put(buf)
	}
}
