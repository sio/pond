//go:build test_cli

package cli

import (
	"github.com/sio/pond/lib/sandbox"
	"testing"

	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var secretctl = fmt.Sprintf("bin/secretctl@%s-%s", runtime.GOOS, runtime.GOARCH)

func TestRepoInitialization(t *testing.T) {
	chdir()
	sandbox := new(sandbox.Sandbox)
	t.Cleanup(sandbox.Cleanup)
	sandbox.Command(secretctl, "-C", "/repo", "init", "tests/keys/master.pub")
	err := sandbox.Build()
	if err != nil {
		t.Fatal(err)
	}
	err = sandbox.Mkdir("/repo", 0777)
	if err != nil {
		t.Fatal(err)
	}

	agent, err := sshAgent(sandbox, "tests/keys/master")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(agent.Stop)

	result, err := sandbox.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("stderr and/or return code check failed:\n%s", result)
	}
	if !strings.Contains(result.Output(), "Initialized new secrets repository: /repo") {
		t.Fatalf("unexpected output after successful execution:\n%s", result)
	}
	if testing.Verbose() {
		t.Logf("\n%s", result)
	}
}

// Switch to top-level Go project directory
func chdir() {
	var path string
	var err error
	path, err = filepath.Abs(".")
	if err != nil {
		panic("abs: " + err.Error())
	}
	for {
		_, err = os.Stat(filepath.Join(path, "go.mod"))
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			panic("stat: " + err.Error())
		}
		if path == "/" {
			panic("could not locate Go project directory")
		}
		path = filepath.Dir(path)
	}
	err = os.Chdir(path)
	if err != nil {
		panic("chdir: " + err.Error())
	}
}
