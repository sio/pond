package main

import (
	"fmt"
	"path/filepath"

	"github.com/sio/pond/initramfs/kmod"
)

type kmodTree struct {
	Kernel string
	Module map[string]kmodInfo
}

type kmodInfo struct {
	Path    string
	Depends []string
}

func parseKmodTree(dir string) (tree kmodTree, err error) {
	tree.Module = make(map[string]kmodInfo)
	err = kmod.Walk(dir, func(path string) error {
		path, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		mod, err := kmod.Info(path)
		if err != nil {
			return err
		}
		if tree.Kernel == "" {
			tree.Kernel = mod.Kernel
		}
		if tree.Kernel != mod.Kernel {
			return fmt.Errorf(
				"kernel release mismatch: building module tree for %s, got %s from %s",
				tree.Kernel,
				mod.Kernel,
				path,
			)
		}
		tree.Module[mod.Name] = kmodInfo{
			Path:    path,
			Depends: mod.Depends,
		}
		return nil
	})
	return tree, err
}

func findKernelModules(dir string, names []string) (modules []srcdest, err error) {
	tree, err := parseKmodTree(dir)
	if err != nil {
		return nil, err
	}
	destDir := "lib/modules/" + tree.Kernel + "/"
	cache := make(map[string]struct{})
	for _, module := range names {
		err = add(module, cache, tree)
		if err != nil {
			return nil, err
		}
	}
	modules = make([]srcdest, 0, len(cache))
	for path := range cache {
		modules = append(modules, srcdest{path, destDir + filepath.Base(path)})
	}
	return modules, nil
}

func add(name string, cache map[string]struct{}, tree kmodTree) error {
	mod, ok := tree.Module[name]
	if !ok {
		return fmt.Errorf("kernel module not found: %s", name)
	}
	_, done := cache[mod.Path]
	if done {
		return nil
	}
	cache[mod.Path] = struct{}{}
	for _, dep := range mod.Depends {
		err := add(dep, cache, tree)
		if err != nil {
			return err
		}
	}
	return nil
}
