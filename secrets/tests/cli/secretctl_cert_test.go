//go:build test_cli

package cli

import (
	"github.com/sio/pond/lib/sandbox"
	"testing"

	"strings"
)

func TestDelegateAdmin(t *testing.T) {
	chdir()
	sandbox := new(sandbox.Sandbox)
	t.Cleanup(sandbox.Cleanup)
	sandbox.Setenv("SECRETS_DIR", "/repo")
	sandbox.Command(secretctl, "init", "tests/keys/master.pub")
	sandbox.Command(secretctl, "cert", "--admin=alice", "--key=tests/keys/alice.pub", "-r", "/users/alice/")
	sandbox.Command(secretctl, "cert", "--admin=alice", "--key=tests/keys/alice.pub", "-rw", "/")
	sandbox.Command(secretctl, "cert", "--user=bob", "--key=tests/keys/bob.pub", "-r", "/users")
	err := sandbox.Build()
	if err != nil {
		t.Fatal(err)
	}
	err = sandbox.Mkdir("/repo", 0777)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := sshAgent(sandbox, "tests/keys/master", "tests/keys/alice")
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
	for _, path := range []string{
		"/repo/access/admin/alice.cert",
		"/repo/access/admin/alice.x01.cert",
		"/repo/access/master.cert",
	} {
		if !sandbox.Exists(path) {
			t.Errorf("Expected path not found in sanbox: %s", path)
		}
	}
	if !strings.Contains(string(result.Output()), "Initialized new secrets repository: /repo") {
		t.Fatalf("unexpected output after successful execution:\n%s", string(result.Output()))
	}
	if testing.Verbose() {
		t.Logf("\n%s", string(result.Output()))
	}

	// Expected to fail
	var fail = [][]string{
		{secretctl, "cert", "--user=bob", "--key=tests/keys/bob.pub", "-rw", "/users"}, // alice has no permission to delegate write access to /users
		{secretctl, "cert", "--user=bob", "--key=tests/keys/bob.pub", "-r", "/something/else"},
	}
	for _, cmd := range fail {
		result, err := sandbox.Run(cmd...)
		if err != nil {
			t.Fatal(err)
		}
		if result.Ok() {
			t.Errorf("expected command to fail, but it exited successfully:\n%s", string(result.Output()))
		} else if testing.Verbose() {
			t.Logf("\n%s\n[exit code %d]", string(result.Output()), result.ExitCode())
		}
	}
}
