package server

import (
	"fmt"
	"sync"
)

// Memory buffers of this size will be frequently used by our NBD server
const BufferSize = 64 << 10

// Reusable byte slice pool
//
// Usage convention:
//   - Do not change capacity (do not move the start of the slice)
//   - Do not assume length (always redefine the end of the slice)
//   - Use .Put() only to return borrowed slices, do not create anything manually
var buffer = bufferPool{
	pool: sync.Pool{
		New: func() any {
			buf := make([]byte, 0, BufferSize)
			return &buf
		},
	},
}

// Even though our pool implementation did not reduce memory allocations
// (1 allocs/op, 24 B/op), it is still beneficial because it catches common
// developer errors
type bufferPool struct {
	pool sync.Pool
}

func (p *bufferPool) Get() (buf []byte) {
	buf = *(p.pool.Get().(*[]byte))
	return buf[:0]
}

func (p *bufferPool) Put(buf []byte) {
	if cap(buf) != BufferSize {
		panic(fmt.Sprintf("attempted to poison the pool: cap=%d, want=%d", cap(buf), BufferSize))
	}
	p.pool.Put(&buf)
}
