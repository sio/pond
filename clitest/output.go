package clitest

import (
	"io"
	"sync"
)

// Buffer output from multiple Writers
//
// Useful for writing stdout and stderr without mangling them together irreversibly
type multiBuffer struct {
	buf    []byte
	chunks []outputChunk
	lock   sync.Mutex
}

type outputChunk struct {
	tag  outputTag
	size int
}

type outputTag uint8

const (
	stdout outputTag = 1
	stderr outputTag = 2
)

func (b *multiBuffer) Writer(tag outputTag) io.Writer {
	return &multiBufferWriter{
		multi: b,
		tag:   tag,
	}
}

func (b *multiBuffer) Write(tag outputTag, data []byte) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.buf = append(b.buf, data...)
	b.chunks = append(b.chunks, outputChunk{tag, len(data)})
}

func (b *multiBuffer) Read(tag outputTag) []byte {
	var out = make([]byte, len(b.buf))
	var outIndex, bufIndex int
	for _, chunk := range b.chunks {
		if chunk.tag == tag {
			copy(out[outIndex:], b.buf[bufIndex:bufIndex+chunk.size])
			outIndex += chunk.size
		}
		bufIndex += chunk.size
	}
	return out[:outIndex]
}

func (b *multiBuffer) ReadAll() []byte {
	return b.buf[:]
}

type multiBufferWriter struct {
	multi *multiBuffer
	tag   outputTag
}

func (w *multiBufferWriter) Write(p []byte) (n int, err error) {
	w.multi.Write(w.tag, p)
	return len(p), nil
}
