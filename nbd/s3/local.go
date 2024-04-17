package s3

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// Local cache backend
type localInterface interface {
	io.ReaderAt
	io.WriterAt
	io.Closer
}

// File based local cache backend
//
// Preallocates full file size ahead of time to reduce fragmentation and
// to avoid running out of disk space unexpectedly.
func openFileBackend(path string, size int64) (localInterface, error) {
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	sys, err := file.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("open syscall connection: %s: %w", file.Name(), err)
	}
	errs := make(chan error, 1)
	defer close(errs)
	err = sys.Control(func(fd uintptr) {
		errs <- syscall.Fallocate(int(fd), 0, 0, size)
	})
	if err != nil {
		return nil, fmt.Errorf("syscall connection: %s: %w", file.Name(), err)
	}
	err = <-errs
	if err != nil {
		return nil, fmt.Errorf("fallocate: %s: %w", file.Name(), err)
	}
	return file, nil
}
