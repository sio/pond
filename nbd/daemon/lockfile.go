package daemon

import (
	"fmt"
	"os"
	"syscall"
)

type Lockfile struct {
	f *os.File
}

func Lock(path string) (*Lockfile, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &Lockfile{f: file}, nil
}

func (l *Lockfile) Close() error {
	err := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	if err != nil {
		return err
	}
	err = l.f.Close()
	if err != nil {
		return err
	}
	err = os.Remove(l.f.Name())
	if err != nil {
		return err
	}
	return nil
}

func (l *Lockfile) String() string {
	var name string
	if l.f != nil {
		name = l.f.Name()
	}
	if len(name) == 0 {
		name = "<nil>"
	}
	return fmt.Sprintf("<Lockfile: %s>", name)
}
