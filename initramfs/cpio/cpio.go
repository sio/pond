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
	"io/fs"
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
		copied: make(map[srcdest]struct{}),
		linked: make(map[srcdest]struct{}),
	}
}

type Archive struct {
	writer   io.Writer
	writerMu sync.Mutex
	dirs     map[string]struct{}
	dirsMu   sync.RWMutex
	copied   map[srcdest]struct{}
	copyMu   sync.RWMutex
	linked   map[srcdest]struct{}
	linkMu   sync.RWMutex
	inode    uint32
}

type srcdest struct {
	src  string
	dest string
}

// Copy local file to cpio archive
func (cpio *Archive) Copy(src, dest string) error {
	if src == dest && len(dest) > 0 && dest[0] == '/' {
		dest = dest[1:] // special case for copying absolute path to the same location
	}
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() && (stat.Mode()&fs.ModeType != fs.ModeSymlink) {
		return fmt.Errorf("not a regular file: %s (%s)", src, stat.Mode())
	}
	cpio.copyMu.RLock()
	_, exists := cpio.copied[srcdest{src, dest}]
	cpio.copyMu.RUnlock()
	if exists {
		return nil
	}
	err = cpio.write(
		file,
		dest,
		Header{
			mode:     modeRegular | uint32(stat.Mode().Perm()),
			filesize: uint32(stat.Size()),
		},
	)
	if err == nil {
		cpio.copyMu.Lock()
		cpio.copied[srcdest{src, dest}] = struct{}{}
		cpio.copyMu.Unlock()
	}
	return err
}

// Create a symbolic link inside cpio archive
func (cpio *Archive) Link(target, linkname string) error {
	cpio.linkMu.RLock()
	_, exists := cpio.linked[srcdest{target, linkname}]
	cpio.linkMu.RUnlock()
	if exists {
		return nil
	}
	err := cpio.write(
		strings.NewReader(target+"\x00"),
		linkname,
		Header{
			mode:     modeSymlink,
			filesize: uint32(len(target) + 1),
		},
	)
	if err == nil {
		cpio.linkMu.Lock()
		cpio.linked[srcdest{target, linkname}] = struct{}{}
		cpio.linkMu.Unlock()
	}
	return err
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
	err := cpio.write(nil, path, Header{mode: modeDirectory | 0755})
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
