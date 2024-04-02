package server

import (
	"sync"
)

// Memory buffers of this size will be frequently used by our NBD server
const BufferSize = 4096

var buffer = sync.Pool{
	New: func() any {
		return make([]byte, 0, BufferSize)
	},
}
