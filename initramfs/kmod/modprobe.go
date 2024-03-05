package kmod

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

var (
	modpath  = make(map[string]string) // short name -> full filesystem path
	modalias []aliasglob
	loaded   = make(map[string]struct{}) // set of already loaded modules
)

// Load kernel module by name
func Load(module string) error {
	_, done := loaded[module]
	if done {
		return nil
	}
	if len(modpath) == 0 {
		refresh()
	}
	path, found := modpath[module]
	if !found {
		return fmt.Errorf("not found: %s", module)
	}
	return loadPath(path)
}

// Load kernel module by modalias
func LoadAlias(alias string) error {
	if len(modalias) == 0 {
		refresh()
	}
	for _, mod := range modalias {
		match, err := filepath.Match(mod.alias, alias)
		if err != nil || !match {
			continue
		}
		err = Load(mod.name)
		if err != nil {
			return err
		}
	}
	return nil
}

// Load kernel module by filepath
func loadPath(path string) error {
	mod, err := Info(path)
	if err != nil {
		return err
	}
	if _, done := loaded[mod.Name]; done {
		return nil
	}
	errs := make([]error, 0)
	for _, dep := range mod.Depends {
		err = Load(dep)
		if err != nil {
			errs = append(errs, err)
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	for err = unix.EBUSY; err == unix.EBUSY; {
		err = unix.FinitModule(int(file.Fd()), "", 0)
	}
	if err != nil &&
		err != unix.EEXIST &&
		err != unix.ENODEV {
		if len(errs) > 0 {
			msg := make([]string, len(errs))
			for i, e := range errs {
				msg[i] = e.Error()
			}
			return fmt.Errorf("could not load dependencies for %s: %s", mod.Name, strings.Join(msg, "; "))
		} else {
			return fmt.Errorf("FinitModule(%s): %w", mod.Name, err)
		}
	}
	loaded[mod.Name] = struct{}{}
	if len(errs) > 0 {
		// Mark all dependencies as successfully loaded if the module itself
		// was loaded correctly.
		// Assume that missing dependencies have been statically built into the
		// running kernel; we know no other way to check
		for _, dep := range mod.Depends {
			loaded[dep] = struct{}{}
		}
	}
	return nil
}

type aliasglob struct {
	name  string
	alias string
}

// Refresh kmod metadata
//
// We do not use modules.alias, modules.dep files because in our initrd
// disk space is more precious than RAM. Number of modules baked into initrd
// is typically very low, so generating this metadata at every boot will not
// become expensive.
func refresh() {
	_ = Walk("", func(path string) error {
		mod, err := Info(path)
		if err != nil {
			return nil
		}
		modpath[mod.Name] = path
		for _, a := range mod.Alias {
			modalias = append(modalias, aliasglob{mod.Name, a})
		}
		return nil
	})
}

// Find all kernel module files in given directory
func Walk(dir string, do func(path string) error) error {
	if dir == "" {
		uname := new(unix.Utsname)
		err := unix.Uname(uname)
		if err != nil {
			return err
		}
		release := string(uname.Release[:bytes.IndexByte(uname.Release[:], 0)])
		dir = "/lib/modules/" + release
	}
	return fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if filepath.Ext(path) != ".ko" {
			return nil
		}
		fullpath := filepath.Join(dir, path)
		stat, err := os.Stat(fullpath)
		if err != nil {
			return nil
		}
		if stat.IsDir() {
			return nil
		}
		if stat.Mode()&0b100100100 == 0 { // check if we have permission to read the file
			return nil
		}
		return do(fullpath)
	})
}
