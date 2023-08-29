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

// Create a directory inside sandbox
func (s *Sandbox) Mkdir(path string, perm os.FileMode) error {
	if s.tmpdir == "" {
		return errNotInitialized
	}
	return os.MkdirAll(filepath.Join(s.tmpdir, path), perm)
}
