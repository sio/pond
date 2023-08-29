// Lightweight sandbox environment
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

// Lightweight sandbox environment for Linux
type Sandbox struct {
	commands [][]string
	files    map[string]none
	env      []string
	tmpdir   string
	chdir    string
	lock     sync.Mutex
	errors   []error
}

// Schedule a command for execution in sandbox
//
// Main executable and any files mentioned in args will be copied to sandbox
// automatically. Shared libraries required by the command will not be copied,
// see CommandWithLibs()
func (s *Sandbox) Command(args ...string) {
	if len(args) == 0 {
		return
	}

	// Add executable
	exe, err := exec.LookPath(args[0])
	if err != nil {
		s.deferError(fmt.Errorf("path lookup for %s: %w", args[0], err))
		return
	}
	s.Add(exe)

	// Add files mentioned on command line
	for _, arg := range args[1:] {
		stat, err := os.Stat(arg)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if stat.IsDir() {
			// Directories mentioned on command line are not copied automatically,
			// it may be too costly and not what caller intended.
			continue
		}
		s.Add(arg)
	}

	// Schedule command for execution
	s.lock.Lock()
	s.commands = append(s.commands, args)
	s.lock.Unlock()
}

// Schedule a command for execution in a sandbox and copy required shared libraries
func (s *Sandbox) CommandWithLibs(args ...string) {
	if len(args) == 0 {
		return
	}
	exe, err := exec.LookPath(args[0])
	if err != nil {
		s.deferError(fmt.Errorf("path lookup for %q: %w", args[0], err))
		return
	}
	libs, err := ldd(exe)
	if err != nil {
		s.deferError(fmt.Errorf("ldd %q: %w", args[0], err))
		return
	}
	s.Add(libs...)
	s.Command(args...)
}

// Add file to the sandbox environment
func (s *Sandbox) Add(filename ...string) {
	if s.files == nil {
		s.lock.Lock()
		s.files = make(map[string]none)
		s.lock.Unlock()
	}
	for _, file := range filename {
		s.files[file] = none{}
	}
}

// Build sandbox environment for executing test commands
//
// Calling this command is not required if no special post-build preparation
// will be performed. (*Sandbox).Execute() will call this automatically at
// startup (if not called previously).
func (s *Sandbox) Build() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.build()
}
func (s *Sandbox) build() error {
	var err error
	if len(s.errors) != 0 {
		return errors.Join(s.errors...)
	}

	s.tmpdir, err = os.MkdirTemp("", "sandbox_*")
	if err != nil {
		return err
	}

	for src := range s.files {
		dest := filepath.Join(s.tmpdir, src)
		if !strings.HasPrefix(dest, s.tmpdir) {
			return fmt.Errorf("possible path traversal attempt: %s -> %s", src, dest)
		}
	}
	for src := range s.files {
		dest := filepath.Join(s.tmpdir, src)
		err = cp(src, dest)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Sandbox) deferError(e error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.errors = append(s.errors, e)
}

// Clean up sandbox environment after executing the test
func (s *Sandbox) Cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.tmpdir) == 0 {
		return
	}
	_ = os.RemoveAll(s.tmpdir)
	s.tmpdir = ""
	s.commands = nil
	s.files = nil
}

type none struct{}
