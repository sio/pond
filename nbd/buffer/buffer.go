// Reusable byte slice pool
//
// Usage convention:
//   - Do not change capacity (do not move the start of the slice)
//   - Do not assume length (always redefine the end of the slice)
//   - Use .Put() only to return borrowed slices, do not create anything manually
package buffer

import (
	"fmt"
	"sync"
)

// Memory buffers of this size will be frequently used by our NBD server
const Size = 64 << 10

// Obtain new buffer from the pool
func Get() []byte {
	return buffer.Get()
}

// Return buffer to the pool after use
func Put(buf []byte) {
	buffer.Put(buf)
}

var buffer = bufferPool{
	pool: sync.Pool{
		New: func() any {
			buf := make([]byte, 0, Size)
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
	if cap(buf) != Size {
		panic(fmt.Sprintf("attempted to poison the pool: cap=%d, want=%d", cap(buf), Size))
	}
	p.pool.Put(&buf)
}
