package clitest

import (
	"fmt"
	"os/exec"
	"strings"
)

var chroot = []string{
	"unshare",
	"--map-root-user",
	"unshare",
	"--mount",
	"--net",
	"--pid",
	"--fork",
}

// Execute test commands one by one until the first failure.
//
// Return error only if something goes terribly wrong.
//
// If one of commands returns non-zero exit code, this function will return nil.
// Use (*Test).ExitCode(), (*Test).Stdout(), (*Test).Stderr(), (*Test).Output()
// to check execution results.
func (t *Test) Execute() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, cmd := range t.commands {
		next, err := t.exec(cmd)
		if err != nil {
			return fmt.Errorf("%v: %w", cmd, err)
		}
		if !next {
			break
		}
	}
	return nil
}

// Execute a single command from test sequence
func (t *Test) exec(command []string) (next bool, err error) {
	var exe = make([]string, len(chroot)+1+len(command))
	copy(exe, chroot)
	exe[len(chroot)] = "--root=" + t.tmpdir
	copy(exe[len(chroot)+1:], command)

	path, err := exec.LookPath(exe[0])
	if err != nil {
		return false, err
	}
	t.output.Write(stdout, []byte(fmt.Sprintf(
		"$ %s\n",
		strings.Join(command, " "),
	)))
	var cmd = exec.Cmd{
		Path:   path,
		Args:   exe,
		Stdout: t.output.Writer(stdout),
		Stderr: t.output.Writer(stderr),
	}
	err = cmd.Run()
	if exit, ok := err.(*exec.ExitError); ok {
		t.exit = new(int)
		*t.exit = exit.ExitCode()
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Read standard output after executing test commands
func (t *Test) Stdout() []byte {
	return t.output.Read(stdout)
}

// Read standard error output after executing test commands
func (t *Test) Stderr() []byte {
	return t.output.Read(stderr)
}

// Read output after executing test commands (stdout + stderr)
func (t *Test) Output() []byte {
	return t.output.ReadAll()
}

// Exit code of test command sequence.
//
// Since execution stops at first failed command, this is either zero if all
// commands were successful or contains the exit code of first failed command.
//
// This function will panic if called before (*Test).Execute()
func (t *Test) ExitCode() int {
	return *t.exit
}
