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
	panic("*Cache.ReadAt not implemented")
}

var (
	_ io.Closer   = new(Cache)
	_ io.ReaderAt = new(Cache)
)
