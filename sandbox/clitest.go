// Test CLI apps in a sandboxed environment.
//
// Works only on Linux, does not require superuser privileges.
// Requires "unshare" from util-linux to be present in $PATH.
package sandbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Initialize new CLI test case (a sequence of commands) to be executed in a
// quasi-chroot environment (no root privileges required)
//
// Binaries corresponding to the provided commands will be added to new rootfs
// automatically. If any of command arguments contains a path to an existing file,
// that file will be copied to new rootfs automatically too.
//
// All other files must be listed explicitly.
// Since target rootfs is by default completely empty, all shared libraries
// must be copied explicitly.
func New(commands [][]string, files []string) (*Test, error) {
	t := &Test{
		commands: commands,
		files:    files,
	}
	err := t.setup()
	if err != nil {
		t.Cleanup()
		return nil, err
	}
	return t, nil
}

// Test runner for command line interfaces
type Test struct {
	commands [][]string
	files    []string
	lock     sync.Mutex
	tmpdir   string
	exit     *int
	output   multiBuffer
}

// Prepare chroot environment for executing test commands
func (t *Test) setup() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	var err error
	t.tmpdir, err = os.MkdirTemp("", "sandbox_*")
	if err != nil {
		return err
	}

	var chroot = make(map[string]struct{})
	for _, cmd := range t.commands {
		if len(cmd) == 0 {
			return fmt.Errorf("encountered empty command in sequence")
		}

		// Copy executables
		exe, err := exec.LookPath(cmd[0])
		if err != nil {
			return err
		}
		chroot[exe] = struct{}{}

		// Copy files mentioned on command line
		for _, arg := range cmd[1:] {
			_, err = os.Stat(arg)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			chroot[arg] = struct{}{}
		}
	}
	for _, file := range t.files {
		// Copy extra files
		chroot[file] = struct{}{}
	}
	for src := range chroot {
		dest := filepath.Join(t.tmpdir, src)
		if !strings.HasPrefix(dest, t.tmpdir) {
			return fmt.Errorf("detected possible chroot escape: %s -> %s", src, dest)
		}
	}
	for src := range chroot {
		dest := filepath.Join(t.tmpdir, src)
		err = cp(src, dest)
		if err != nil {
			return err
		}
	}
	return nil
}

// Clean up chroot environment after executing the test
func (t *Test) Cleanup() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.tmpdir) == 0 {
		return
	}
	_ = os.RemoveAll(t.tmpdir)
	t.tmpdir = ""
	t.output = multiBuffer{}
	t.exit = nil
}
