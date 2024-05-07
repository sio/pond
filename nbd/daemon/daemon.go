package daemon

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/sio/pond/nbd/logger"
	"github.com/sio/pond/nbd/s3"
	"github.com/sio/pond/nbd/server"
)

type Daemon struct {
	S3 struct {
		Endpoint string
		Bucket   string
		Prefix   string
		Access   string
		Secret   string
	}
	Cache struct {
		Dir string
	}
	Listen []struct {
		Network string
		Address string
	}
}

func (d *Daemon) Run() error {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(server.NBD_ESHUTDOWN)
	log := logger.FromContext(ctx)

	// Exclusive lock on local cache directory
	abs, err := filepath.Abs(d.Cache.Dir)
	if err == nil {
		d.Cache.Dir = abs
	}
	lock, err := Lock(filepath.Join(d.Cache.Dir, "lock"))
	if err != nil {
		return fmt.Errorf("acquire cache directory lock: %w", err)
	}
	defer func() {
		err := lock.Close()
		if err != nil {
			log.Error("failed to release the lock file", "lock", lock, "error", err)
		}
	}()

	// Cache object memoization
	var (
		volume   = make(map[string]server.Backend)
		volumeMu sync.Mutex
	)
	export := func(name string) (server.Backend, error) {
		volumeMu.Lock()
		defer volumeMu.Unlock()

		cache, found := volume[name]
		if found {
			return &dontClose{r: cache}, nil
		}
		cache, err := s3.Open(
			d.S3.Endpoint,
			d.S3.Access,
			d.S3.Secret,
			d.S3.Bucket,
			filepath.Join(d.S3.Prefix, name),
			d.Cache.Dir,
		)
		if err != nil {
			return nil, err
		}
		// TODO: clean up old cache artifacts when running low on disk space
		volume[name] = cache
		return &dontClose{r: cache}, nil
	}

	// Launch NBD server
	nbd := server.New(ctx, export)
	go nbd.ListenShutdown()
	var group errgroup.Group
	for _, listener := range d.Listen {
		listener := listener
		group.Go(func() error {
			err := nbd.Listen(listener.Network, listener.Address)
			if err != nil {
				log.Error("nbd listener failed", "listener", fmt.Sprintf("%s://%s", listener.Network, listener.Address), "error", err)
			}
			return err
		})
	}
	err = group.Wait()
	for name, cache := range volume {
		closer, ok := cache.(io.Closer)
		if !ok {
			continue
		}
		e := closer.Close()
		if e != nil {
			log.Error("closing cache failed", "name", name, "error", e)
		}
	}
	return err
}

// Hide Close() method from type assertion to avoid accidental closing of
// memoized cache objects
type dontClose struct {
	r io.ReaderAt
}

func (r *dontClose) ReadAt(p []byte, offset int64) (int, error) {
	return r.r.ReadAt(p, offset)
}
