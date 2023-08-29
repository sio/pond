//go:build test_cli

package cli

import (
	"github.com/sio/pond/lib/sandbox"
	"testing"

	"fmt"
	"os"
	"runtime"
	"strings"
)

var secretctl = fmt.Sprintf("bin/secretctl@%s-%s", runtime.GOOS, runtime.GOARCH)

func TestRepoInitialization(t *testing.T) {
	os.Chdir("../..")

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
		t.Logf("Exit code: %d", result.ExitCode())
		t.Logf("Output:\n%s", string(result.Output()))
		t.FailNow()
	}
	if !strings.Contains(string(result.Output()), "Initialized new secrets repository: /repo") {
		t.Fatalf("unexpected output after successful execution:\n%s", string(result.Output()))
	}
	if testing.Verbose() {
		t.Logf("\n%s", string(result.Output()))
	}
}
