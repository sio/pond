// Read cache for S3 objects that are assumed to be immutable
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sio/pond/nbd/buffer"
	"github.com/sio/pond/nbd/logger"
	"github.com/sio/pond/nbd/verity"
)

type Cache struct {
	// Connection to remote S3 object
	remote remoteInterface

	// Local backend for cached object
	local localInterface

	// Chunk availability map
	chunk *chunkMap

	// Network connection limiter
	queue *Queue

	// Top level context
	ctx    context.Context
	cancel context.CancelCauseFunc

	// Time of the last cache miss
	atime atomic.Value

	// Keep track of spawned goroutines
	goro *sync.WaitGroup
}

func Open(endpoint, access, secret, bucket, object, localdir string) (c *Cache, err error) {
	c = new(Cache)
	c.ctx, c.cancel = context.WithCancelCause(context.TODO())
	c.ctx = logger.With(c.ctx, "s3", fmt.Sprintf("%s/%s/%s", endpoint, bucket, object))
	c.goro = new(sync.WaitGroup)
	c.queue = NewQueue(c.ctx, connLimitPerObject)
	c.atime.Store(time.Now())
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
		defer c.goro.Done()
		c.chunk.AutoSave(c.ctx)
	}()

	c.goro.Add(1)
	go func() {
		defer c.goro.Done()
		c.bgFetchAll()
	}()

	c.goro.Add(1)
	go func() {
		defer c.goro.Done()
		checksum, err := verity.Open(c)
		if err != nil {
			log := logger.FromContext(c.ctx)
			log.Info("background data integrity validation disabled", "error", err)
			return
		}
		c.bgIntegrity(checksum)
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
		c.queue,
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
		go func(part chunk) {
			defer c.goro.Done()
			err := c.fetch(part, false)
			if err != nil {
				cancel(err)
			}
		}(part)
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
func (c *Cache) fetch(part chunk, background bool) (err error) {
	wait, done := c.chunk.Check(part)
	if done {
		return nil
	}

	ctx, cancel := context.WithCancelCause(c.ctx)
	defer cancel(errNotRelevant)

	c.goro.Add(1)
	go func() {
		defer c.goro.Done()
		select {
		case <-wait:
			cancel(errDoneElsewhere)
		case <-ctx.Done():
		}
	}()

	if background {
		err = AcquireLowPriority(ctx, globalConnectionQueue, c.queue)
	} else {
		err = Acquire(ctx, globalConnectionQueue, c.queue)
	}
	if err != nil {
		_, done = c.chunk.Check(part)
		if done {
			return nil
		}
		return err
	}

	if !background {
		c.atime.Store(time.Now())
	}

	offset, size := c.chunk.Offset(part)
	remote, err := c.remote.Reader(ctx, offset, int64(size))
	if err != nil {
		return err
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
		return err
	}
	if n != int64(size) {
		return fmt.Errorf("%w: written %d bytes, want %d bytes", io.ErrShortWrite, n, size)
	}
	c.chunk.Done(part)
	return nil
}

// Fetch all data from remote to local storage (warm up the cache)
func (c *Cache) bgFetchAll() {
	const (
		// Do nothing if there was higher priority activity recently
		idleDelay = 1 * time.Minute

		// Retry each block N times before giving up
		retryLimit = 5
	)
	var retry int
	var part chunk
	for uint64(part)*chunkSize < c.chunk.size {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(time.Until(c.atime.Load().(time.Time).Add(idleDelay))):
		}
		if time.Since(c.atime.Load().(time.Time)) < idleDelay {
			continue
		}
		err := c.fetch(part, true)
		if err != nil && retry < retryLimit {
			retry++
			continue
		}
		if err != nil {
			log := logger.FromContext(c.ctx)
			log.Warn("background fetch failed", "chunk", part, "error", err)
		}
		retry = 0 // reset counter on success or after giving up on bad chunk
		part++
	}
}

// Check data integrity in background indefinitely
func (c *Cache) bgIntegrity(checksum verity.Verity) {
	// Delay before the first scrub
	sleep := time.After(time.Hour + rand.N(10*time.Minute))

	// Delay before each consecutive scrub after that
	delayBetweenScrubs := 3*time.Hour + rand.N(10*time.Minute)

	log := logger.FromContext(c.ctx)
	var part chunk
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-sleep:
		}
		offset, size := c.chunk.Offset(part)
		err := checksum.Verify(c, offset, size)
		if err != nil {
			log.Error("integrity verification failed", "chunk", part, "error", err)
			c.chunk.Lost(part)
		}
		var ok bool
		part, ok = c.chunk.After(part)
		if !ok {
			// No more chunks to check, wait for a while and start from the beginning
			part = 0
			sleep = time.After(delayBetweenScrubs)
		}
	}
}

var (
	_ io.Closer   = new(Cache)
	_ io.ReaderAt = new(Cache)
)

var (
	errNotRelevant   = errors.New("context not relevant anymore")
	errDoneElsewhere = errors.New("work completed by a concurrent goroutine")
)
