package server

import (
	"sync"
)

// Memory buffers of this size will be frequently used by our NBD server
const BufferSize = 4096

// Reusable byte slice pool
//
// Usage convention:
//   - Do not change capacity (do not move the start of the slice)
//   - Do not assume length (always redefine the end of the slice)
//   - Use .Put() only to return borrowed slices, do not create anything manually
var buffer = sync.Pool{
	New: func() any {
		return make([]byte, 0, BufferSize)
	},
}
