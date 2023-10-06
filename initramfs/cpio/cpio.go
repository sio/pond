// Extremely basic write-only implementation of cpio archive format
//
// Stores only regular files and directories, all owned by root.
// Modification times are not preserved. Symlinks are always dereferenced.
//
// See `man 5 cpio` for format description: <https://manpages.debian.org/5/cpio>
package cpio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	modeRegular   uint32 = 0100000
	modeDirectory uint32 = 0040000
	modeSymlink   uint32 = 0120000
	pathSeparator        = "/"
)

var magicNewAsciiNoChecksum = [3]byte{07, 07, 01}

// Provide cpio wrapper for given io.Writer
func New(w io.Writer) *Archive {
	return &Archive{
		writer: w,
		dirs:   make(map[string]struct{}),
	}
}

type Archive struct {
	writer   io.Writer
	writerMu sync.Mutex
	dirs     map[string]struct{}
	dirsMu   sync.RWMutex
	inode    uint32
}

// Copy local file to cpio archive
func (cpio *Archive) Copy(src, dest string) error {
	header, err := fileHeader(src)
	if err != nil {
		return err
	}
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	err = cpio.write(file, dest, header)
	if err != nil {
		return err
	}
	return nil
}

// Create a symbolic link inside cpio archive
func (cpio *Archive) Link(target, linkname string) error {
	return cpio.write(
		strings.NewReader(target+"\x00"),
		linkname,
		Header{
			mode:     modeSymlink,
			filesize: uint32(len(target) + 1),
		},
	)
}

func (cpio *Archive) write(data io.Reader, path string, header Header) error {
	if len(path) == 0 {
		return errors.New("empty path in archive")
	}
	if path[:1] == pathSeparator {
		return errors.New("absolute paths in archive are not supported")
	}
	path = filepath.ToSlash(filepath.Clean(path))
	elements := strings.Split(path, pathSeparator)
	for i := 1; i < len(elements); i++ {
		dir := strings.Join(elements[:i], pathSeparator)
		err := cpio.mkdir(dir)
		if err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	cpio.writerMu.Lock()
	defer cpio.writerMu.Unlock()

	header.inode = cpio.inode
	cpio.inode++
	err := header.Write(cpio.writer, path)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}
	written, err := io.Copy(cpio.writer, data)
	if err != nil {
		return err
	}
	if written != int64(header.filesize) {
		return fmt.Errorf("written data size (%dB) does not match header value (%dB): %s", written, header.filesize, path)
	}
	const align = 4
	padding := header.filesize % align
	if padding != 0 {
		padding = align - padding
	}
	var zero [align]byte
	_, err = cpio.writer.Write(zero[:padding])
	if err != nil {
		return err
	}
	return nil
}

func (cpio *Archive) mkdir(path string) error {
	cpio.dirsMu.RLock()
	_, exists := cpio.dirs[path]
	cpio.dirsMu.RUnlock()
	if exists {
		return nil
	}
	header := Header{
		mode: modeDirectory | 0755,
	}
	err := cpio.write(nil, path, header)
	if err != nil {
		return err
	}
	cpio.dirsMu.Lock()
	cpio.dirs[path] = struct{}{}
	cpio.dirsMu.Unlock()
	return nil
}

// Finalize cpio archive
func (cpio *Archive) Close() error {
	return cpio.write(nil, "TRAILER!!!", Header{})
}
