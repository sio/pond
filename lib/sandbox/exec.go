package sandbox

import (
	"fmt"
	"os/exec"
	"strings"
)

var unshare = []string{
	"unshare",
	"--map-root-user",
	"unshare",
	"--mount",
	"--net",
	"--pid",
	"--fork",
	"--wd=/",
	"--root=",
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

// Change working directory for sandbox commands
func (s *Sandbox) Chdir(path string) {
	s.chdir = path
}

// Execute a single command not scheduled beforehand.
// Assumes that sandbox was already built.
//
// Useful for running customizable diagnostic tasks in addition to the
// scheduled commands, and for running commands that are expected to fail.
func (s *Sandbox) Run(args ...string) (*Result, error) {
	if s.tmpdir == "" {
		return nil, fmt.Errorf("sandbox not initialized")
	}
	result := new(Result)
	_, err := s.exec(args, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Execute a single command from test sequence
func (s *Sandbox) exec(command []string, result *Result) (next bool, err error) {
	var args = make([]string, len(unshare)+len(command))
	copy(args, unshare)
	copy(args[len(unshare):], command)
	if s.chdir != "" {
		err = s.Mkdir(s.chdir, 0777)
		if err != nil {
			return false, err
		}
		args[len(unshare)-2] = "--wd=" + s.chdir
	}
	args[len(unshare)-1] = "--root=" + s.tmpdir

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
		Env:    s.env,
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

func (r *Result) String() string {
	if r == nil {
		return "<nil>"
	}
	output := strings.TrimRight(r.Output(), "\n\r \t")
	if r.ExitCode() == 0 {
		return output
	}
	return fmt.Sprintf("%s\n[exit code %d]", output, r.ExitCode())
}

// Read standard output
func (r *Result) Stdout() string {
	return string(r.output.Read(stdout))
}

// Read standard error
func (r *Result) Stderr() string {
	return string(r.output.Read(stderr))
}

// Read all output (stdout and stderr)
func (r *Result) Output() string {
	return string(r.output.ReadAll())
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
