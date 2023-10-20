package ldd

import (
	"bytes"
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/unix"
)

var ldcache *cache
var arch string

// Return shared object dependencies for an executable at provided path
func Depends(path string) ([]string, error) {
	exe, err := elf.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exe.Close() }()
	var deps = make(map[string]struct{})

	interp, err := interpreter(exe)
	if err == nil {
		deps[interp] = struct{}{}
		interpDeps, err := Depends(interp)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch indirect dependencies (%s): %w", interp, err)
		}
		for _, p := range interpDeps {
			deps[p] = struct{}{}
		}
	}

	libs, err := exe.ImportedLibraries()
	if err != nil {
		return nil, err
	}
	for _, lib := range libs {
		lib, err = library(lib)
		if err != nil {
			return nil, err
		}
		deps[lib] = struct{}{}
	}
	var result = make([]string, 0, len(deps))
	for dep := range deps {
		result = append(result, dep)
	}
	return result, nil
}

// Resolve library path by name
func library(lib string) (path string, err error) {
	if ldcache == nil {
		ldcache = newCache()
	}
	if arch == "" {
		arch = machineArch()
	}
	path, found := ldcache.Get(lib)
	if found {
		return path, nil
	}
	dirs := [...]string{"/lib", "/usr/lib", "/lib64", "/usr/lib64"}
	gnudir := arch + "-linux-gnu"
	for _, dir := range dirs {
		path := filepath.Join(dir, lib)
		_, err := os.Stat(path)
		if err != nil {
			path = filepath.Join(dir, gnudir, lib)
			_, err = os.Stat(path)
		}
		if err == nil {
			ldcache.Set(lib, path)
			return path, nil
		}
	}
	return "", fmt.Errorf("failed to resolve library path: %s", lib)
}

func interpreter(exe *elf.File) (path string, err error) {
	section := exe.Section(".interp")
	if section == nil {
		return "", errors.New("section not found: .interp")
	}
	raw, err := section.Data()
	if err != nil {
		return "", err
	}
	return string(raw[:bytes.IndexByte(raw, 0)]), nil
}

type cache struct {
	item map[string]string
	mu   sync.RWMutex
}

func newCache() *cache {
	ldcache, err := ldsoCache("")
	if err != nil {
		ldcache = make(map[string]string)
	}
	return &cache{item: ldcache}
}

func (c *cache) Set(key, value string) {
	c.mu.Lock()
	c.item[key] = value
	c.mu.Unlock()
}

func (c *cache) Get(key string) (value string, found bool) {
	c.mu.RLock()
	value, found = c.item[key]
	c.mu.RUnlock()
	return value, found
}

func machineArch() string {
	var uname = new(unix.Utsname)
	err := unix.Uname(uname)
	if err != nil {
		return "unknown_arch"
	}
	return string(uname.Machine[:bytes.IndexByte(uname.Machine[:], 0)])
}
