package audit

import (
	"fmt"
	"os"
	"time"
)

// A primitive filesystem based lock
//
// Since no OS-level locking is happening here (for simplicity of cross
// platform support), this lock is vulnerable to race conditions when multiple
// processes try to acquire it simultaneously.
// This is not a problem for our intended use case.
type lockfile struct {
	file *os.File
	path string
}

func (lock *lockfile) TryLock(filename string) error {
	if lock.path != "" || lock.file != nil {
		return fmt.Errorf("lock struct reuse: bad attempt, call Unlock() first")
	}
	start := time.Now()
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("stat: %w", err)
	}
	delta := start.Sub(stat.ModTime())
	if delta > 0 {
		_ = file.Close()
		return fmt.Errorf("lock file created %s before locking: %s", delta, filename)
	}
	*lock = lockfile{
		file: file,
		path: filename,
	}
	_, err = fmt.Fprintf(file, "Locked by PID %s at %s\n", os.Getpid(), time.Now())
	if err != nil {
		lock.Unlock()
		return fmt.Errorf("write to lock file: %w", err)
	}
	return nil
}

func (lock *lockfile) Unlock() {
	if lock.file != nil {
		_ = lock.file.Close()
	}
	lock.file = nil
	if lock.path != "" {
		_ = os.Remove(lock.path)
	}
	lock.path = ""
}
