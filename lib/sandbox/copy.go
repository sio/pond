package sandbox

import (
	"io"
	"os"
	"path/filepath"
)

// Copy a file while creating all required directories for destination
func cp(src, dest string) error {
	var from, to *os.File
	var err error
	from, err = os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = from.Close() }()
	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return err
	}
	stat, err := from.Stat()
	if err != nil {
		return err
	}
	to, err = os.OpenFile(dest, os.O_RDWR|os.O_CREATE, stat.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = to.Close() }()
	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}
	err = from.Close()
	if err != nil {
		return err
	}
	err = to.Close()
	if err != nil {
		return err
	}
	return nil
}
