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
	search := recursive{}
	err := search.Search(path)
	if err != nil {
		return nil, err
	}
	var result = make([]string, 0, len(search.found))
	for dep := range search.found {
		result = append(result, dep)
	}
	return result, nil
}

type recursive struct {
	visited map[string]struct{}
	found   map[string]struct{}
}

func (s *recursive) Search(path string) error {
	if s.visited == nil {
		s.visited = make(map[string]struct{})
	}
	if s.found == nil {
		s.found = make(map[string]struct{})
	}

	if _, seen := s.visited[path]; seen {
		return nil
	}
	s.visited[path] = struct{}{}

	exe, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = exe.Close() }()

	interp, err := interpreter(exe)
	if err == nil {
		s.found[interp] = struct{}{}
		err = s.Search(interp)
		if err != nil {
			return fmt.Errorf("%s: %w", interp, err)
		}
	}
	dirs, err := exe.DynString(elf.DT_RUNPATH)
	if err != nil {
		return fmt.Errorf("%s: failed to parse DT_RUNPATH", path)
	}
	libs, err := exe.ImportedLibraries()
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	for _, lib := range libs {
		libpath, err := library(lib, dirs)
		if err != nil {
			return fmt.Errorf("%s: %w", lib, err)
		}
		s.found[libpath] = struct{}{}
		err = s.Search(libpath)
		if err != nil {
			return fmt.Errorf("%s: %w", libpath, err)
		}
	}
	return nil
}

// Resolve library path by name
func library(lib string, dirs []string) (path string, err error) {
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
	dirs = append(dirs, "/lib", "/usr/lib", "/lib64", "/usr/lib64")
	gnudir := arch + "-linux-gnu"
	origLen := len(dirs)
	for i := 0; i < origLen; i++ {
		dirs = append(dirs, filepath.Join(dirs[i], gnudir))
	}
	for _, dir := range dirs {
		path := filepath.Join(dir, lib)
		_, err := os.Stat(path)
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
