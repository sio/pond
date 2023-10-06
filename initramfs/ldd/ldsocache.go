package ldd

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	cacheMagic       = "glibc-ld.so.cache"
	cacheVersion     = "1.1"
	cacheDefaultPath = "/etc/ld.so.cache"
)

// /etc/ld.so.cache is a binary file created by glibc tools
func ldsoCache(filename string) (map[string]string, error) {
	if len(filename) == 0 {
		filename = cacheDefaultPath
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	var header cacheHeader
	err = binary.Read(file, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	err = header.Validate()
	if err != nil {
		return nil, err
	}
	_, err = file.Seek(-int64(header.TableSize), io.SeekEnd)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(splitNuls)
	var cache = make(map[string]string)
	var key string
	var readKey bool
	for scanner.Scan() {
		readKey = !readKey
		if readKey {
			key = scanner.Text()
			continue
		}
		cache[key] = scanner.Text()
		if !strings.HasSuffix(cache[key], key) {
			return nil, fmt.Errorf("mismatching cache value: %s => %s", key, cache[key])
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	if len(cache) != int(header.Count) {
		return nil, fmt.Errorf("mismatching entries count: header advertised %d items, got %d", header.Count, len(cache))
	}
	return cache, nil
}

// More information about header struct:
// - https://sourceware.org/git/?p=glibc.git;a=blob;f=sysdeps/generic/dl-cache.h;hb=a3c50bf46a1ca6d9d2b7d879176d345abf95a9de#l157
// - https://github.com/chainguard-dev/ldso-cache/blob/main/ldsocache.go
type cacheHeader struct {
	Magic     [17]byte
	Version   [3]byte
	Count     uint32
	TableSize uint32
	_         uint8
	_         [3]byte
	_         uint32
	_         [3]uint32
}

func (h *cacheHeader) Validate() error {
	if string(h.Magic[:]) != cacheMagic {
		return fmt.Errorf("unsupported magic value: %s", h.Magic)
	}
	if string(h.Version[:]) != cacheVersion {
		return fmt.Errorf("unsupported %s version: %s", h.Magic, h.Version)
	}
	return nil
}

func splitNuls(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, 0); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
