package sandbox

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

// Execute scheduled commands one by one until the first failure.
//
// Return error only if something goes terribly wrong, use (*Result) to check
// commands status and output.
func (s *Sandbox) Execute() (*Result, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.tmpdir == "" {
		err := s.build()
		if err != nil {
			return nil, err
		}
	}

	result := new(Result)
	for _, cmd := range s.commands {
		next, err := s.exec(cmd, result)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", cmd, err)
		}
		if !next {
			break
		}
	}
	return result, nil
}

// Execute a single command from test sequence
func (s *Sandbox) exec(command []string, result *Result) (next bool, err error) {
	var args = make([]string, len(chroot)+1+len(command))
	copy(args, chroot)
	args[len(chroot)] = "--root=" + s.tmpdir
	copy(args[len(chroot)+1:], command)

	path, err := exec.LookPath(args[0])
	if err != nil {
		return false, err
	}
	result.output.Write(stdout, []byte(fmt.Sprintf(
		"$ %s\n",
		strings.Join(command, " "),
	)))
	var cmd = exec.Cmd{
		Path:   path,
		Args:   args,
		Stdout: result.output.Writer(stdout),
		Stderr: result.output.Writer(stderr),
	}
	err = cmd.Run()
	if exit, ok := err.(*exec.ExitError); ok {
		result.exit = exit.ExitCode()
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Result of executing commands in sandbox
type Result struct {
	output multiBuffer
	exit   int
}

// Read standard output
func (r *Result) Stdout() []byte {
	return r.output.Read(stdout)
}

// Read standard error
func (r *Result) Stderr() []byte {
	return r.output.Read(stderr)
}

// Read all output (stdout and stderr)
func (r *Result) Output() []byte {
	return r.output.ReadAll()
}

// Exit code of test command sequence.
//
// Since execution stops at first failed command, this is either zero if all
// commands were successful or contains the exit code of first failed command.
func (r *Result) ExitCode() int {
	return r.exit
}

// Check if all commands executed without errors
func (r *Result) Ok() bool {
	return r.exit == 0 && len(r.Stderr()) == 0
}
