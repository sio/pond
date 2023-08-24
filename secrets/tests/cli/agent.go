//go:build test_cli

package cli

import (
	"sandbox"

	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

// Start ssh-agent and load private keys
func sshAgent(s *sandbox.Sandbox, key ...string) (*agent, error) {
	const innerSocket = "/ssh-agent.sock"
	s.Setenv("SSH_AUTH_SOCK", innerSocket)
	socket, err := s.Path(innerSocket)
	if err != nil {
		return nil, err
	}
	agent := &agent{
		socket: socket,
	}
	err = agent.Start()
	if err != nil {
		agent.Stop()
		return nil, err
	}
	for _, k := range key {
		err = agent.Add(k)
		if err != nil {
			agent.Stop()
			return nil, err
		}
	}
	return agent, nil
}

// ssh-agent process that is accessible both inside and outside the sandbox
type agent struct {
	socket string
	pid    int
}

var agentPidRegex = regexp.MustCompile(`SSH_AGENT_PID=(\d+)`)

func (a *agent) Start() error {
	launcher := exec.Command("ssh-agent", "-a", a.socket)
	var output = new(bytes.Buffer)
	launcher.Stdout = output
	launcher.Stderr = output
	err := launcher.Run()
	if err != nil {
		return err
	}
	match := agentPidRegex.FindSubmatch(output.Bytes())
	if len(match) < 2 {
		return errors.New("SSH_AGENT_PID not found in launcher output")
	}
	a.pid, err = strconv.Atoi(string(match[1]))
	if err != nil {
		return err
	}
	return nil
}

func (a *agent) Stop() {
	if a.pid == 0 {
		return
	}
	process, err := os.FindProcess(a.pid)
	if err != nil {
		return
	}
	_ = process.Kill()
	a.pid = 0
}

func (a *agent) Add(keypath string) error {
	// Test keys are often world-readable. Fix that silently
	stat, err := os.Stat(keypath)
	if err != nil {
		return err
	}
	const (
		visible = 0077
		private = 0700
	)
	if stat.Mode()&visible != 0 {
		err = os.Chmod(keypath, stat.Mode()&private)
		if err != nil {
			return err
		}
	}

	// Add key to ssh-agent
	add := exec.Command("ssh-add", keypath)
	add.Env = append(
		os.Environ(),
		fmt.Sprintf("SSH_AUTH_SOCK=%s", a.socket),
	)
	var output = new(bytes.Buffer)
	add.Stdout = output
	add.Stderr = output
	err = add.Run()
	if err != nil {
		return fmt.Errorf("%w:\n%s", err, output.String())
	}
	return nil
}
