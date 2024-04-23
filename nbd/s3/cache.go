// Read cache for S3 objects that are assumed to be immutable
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/sio/pond/nbd/buffer"
)

type Cache struct {
	// Connection to remote S3 object
	remote remoteInterface

	// Local backend for cached object
	local localInterface

	// Chunk availability map
	chunk *chunkMap

	// Top level context
	ctx    context.Context
	cancel context.CancelCauseFunc

	// Keep track of spawned goroutines
	goro *sync.WaitGroup
}

func Open(endpoint, access, secret, bucket, object, localdir string) (c *Cache, err error) {
	c = new(Cache)
	c.ctx, c.cancel = context.WithCancelCause(context.TODO())
	c.goro = new(sync.WaitGroup)
	c.remote, err = openMinioRemote(endpoint, access, secret, bucket, object)
	if err != nil {
		return nil, fmt.Errorf("open remote: %w", err)
	}
	c.local, err = openFileBackend(filepath.Join(localdir, object), c.remote.Size())
	if err != nil {
		return nil, fmt.Errorf("open local backend: %w", err)
	}
	c.chunk, err = openChunkMap(filepath.Join(localdir, object+".chunk"), c.remote.Size())
	if err != nil {
		return nil, fmt.Errorf("open chunk map: %w", err)
	}
	c.goro.Add(1)
	go func() {
		c.chunk.AutoSave(c.ctx)
		c.goro.Done()
	}()
	return c, nil
}

func (c *Cache) Close() error {
	c.cancel(fmt.Errorf("cache closed"))
	errs := make([]error, 0)
	for _, component := range []io.Closer{
		c.remote,
		c.local,
		c.chunk,
	} {
		errs = append(errs, component.Close())
	}
	wait, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()
	go func() {
		<-wait.Done()
		if errors.Is(context.Cause(wait), context.DeadlineExceeded) {
			panic("orphaned goroutines left behind after closing cache object")
		}
	}()
	c.goro.Wait()
	return errors.Join(errs...)
}

func (c *Cache) ReadAt(p []byte, offset int64) (n int, err error) {
	ctx, cancel := context.WithCancelCause(c.ctx)
	defer cancel(errNotRelevant)

	// Schedule relevant chunks to be fetched
	part := chunk(offset / chunkSize)
	for remain := int64(len(p)); remain > 0; remain -= chunkSize {
		c.goro.Add(1)
		go c.fetch(part, cancel)
		part++
	}

	// Return data from the first relevant chunk
	ready, _ := c.chunk.Check(chunk(offset / chunkSize))
	select {
	case <-ready:
		return c.local.ReadAt(p[:min(len(p), chunkSize)], offset)
	case <-ctx.Done():
		return 0, context.Cause(ctx)
	}
}

// This function intentionally uses a context independent from ReadAt:
// even if caller was cancelled it is still useful to finish caching the current
// chunk for future use.
func (c *Cache) fetch(part chunk, fail context.CancelCauseFunc) {
	defer c.goro.Done()
	wait, done := c.chunk.Check(part)
	if done {
		return
	}

	ctx, cancel := context.WithCancelCause(c.ctx)
	defer cancel(errNotRelevant)

	// Kill both current chunk context and parent ReadAt context
	// in case of irrecoverable errors
	fatal := func(e error) {
		cancel(e)
		fail(e)
	}

	c.goro.Add(1)
	go func() {
		select {
		case <-wait:
			cancel(errDoneElsewhere)
		case <-ctx.Done():
		}
		c.goro.Done()
	}()

	offset, size := c.chunk.Offset(part)
	remote, err := c.remote.Reader(ctx, offset, size)
	if err != nil {
		fatal(err)
		return
	}
	defer func() {
		err := remote.Close()
		if err != nil {
			cancel(err) // only affects fetch() and not ReadAt()
		}
	}()

	buf := buffer.Get()
	defer buffer.Put(buf)

	n, err := io.CopyBuffer(io.NewOffsetWriter(c.local, offset), remote, buf[:cap(buf)])
	if err != nil {
		fatal(err)
		return
	}
	if n != size {
		fatal(fmt.Errorf("%w: written %d bytes, want %d bytes", io.ErrShortWrite, n, size))
		return
	}
	c.chunk.Done(part)
}

var (
	_ io.Closer   = new(Cache)
	_ io.ReaderAt = new(Cache)
)

var (
	errNotRelevant   = errors.New("context not relevant anymore")
	errDoneElsewhere = errors.New("work completed by a concurrent goroutine")
)
