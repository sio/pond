package agent

import (
	"testing"

	"sync/atomic"
)

// This benchmark shows that adding an atomic counter to Sign() method of
// ssh-agent will not negatively affect performance: atomic adds are six orders
// of magnitude faster than ssh-agent socket calls (8ns vs 2-8ms)
//
// ssh-agent performance is measured with <https://github.com/sio/ssh-agent-benchmark>
func BenchmarkAtomic(b *testing.B) {
	var counter uint32
	for i := 0; i < b.N; i++ {
		atomic.AddUint32(&counter, 1)
	}
	if counter != uint32(b.N) {
		b.Fatalf("counter=%d, expected=%d", counter, b.N)
	}
}
