package sandbox

import (
	"errors"
	"os"
	"path/filepath"
)

var errNotInitialized = errors.New("sandbox file system not created yet, call Build() first")

// Get path to a file inside sandbox
func (s *Sandbox) Path(inside string) (outside string, err error) {
	if s.tmpdir == "" {
		return "", errNotInitialized
	}
	return filepath.Join(s.tmpdir, inside), nil
}

// Check if a path exists inside sandbox
func (s *Sandbox) Exists(path string) bool {
	if s.tmpdir == "" {
		return false
	}
	path = filepath.Join(s.tmpdir, path)
	_, err := os.Stat(path)
	return err == nil
}

// Create a directory inside sandbox
func (s *Sandbox) Mkdir(path string, perm os.FileMode) error {
	if s.tmpdir == "" {
		return errNotInitialized
	}
	return os.MkdirAll(filepath.Join(s.tmpdir, path), perm)
}
