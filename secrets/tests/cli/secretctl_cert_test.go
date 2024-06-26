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
		t.Fatalf("stderr and/or return code check failed:\n%s", result)
	}
	for _, path := range []string{
		"/repo/access/admin/alice.01.cert",
		"/repo/access/admin/alice.02.cert",
		"/repo/access/user/bob.01.cert",
		"/repo/access/master.cert",
	} {
		if !sandbox.Exists(path) {
			t.Errorf("Expected path not found in sandbox: %s", path)
		}
	}
	if !strings.Contains(result.Output(), "Initialized new secrets repository: /repo") {
		t.Fatalf("unexpected output after successful execution:\n%s", result)
	}
	if testing.Verbose() {
		t.Logf("\n%s", result)
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
			t.Errorf("expected command to fail, but it exited successfully:\n%s", result)
		} else if testing.Verbose() {
			t.Logf("\n%s", result)
		}
	}

	// Remove (=revoke) first certificate
	err = sandbox.Remove("/repo/access/admin/alice.01.cert")
	if err != nil {
		t.Fatalf("removing alice.01.cert: %v", err)
	}
	result, err = sandbox.Run(secretctl, "cert", "--user=bob", "--key=tests/keys/bob.pub", "-r", "/users")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ok() {
		t.Fatalf("issuing certificate succeeded after admin cert revocation:\n%s", result)
	} else if testing.Verbose() {
		t.Logf("\n%s", result)
	}

	// Issue a certificate for alice after one of previous certs was revoked
	result, err = sandbox.Run(secretctl, "cert", "--admin=alice", "--key=tests/keys/alice.pub", "-r", "/users/alice/")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() || testing.Verbose() {
		t.Logf("\n%s\n", result)
	}
	if !result.Ok() || !strings.Contains(result.Output(), "alice.03.cert") {
		t.FailNow()
	}
}
