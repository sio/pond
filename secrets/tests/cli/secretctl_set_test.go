//go:build test_cli

package cli

import (
	"github.com/sio/pond/lib/sandbox"
	"testing"
)

func TestSecretctlSet(t *testing.T) {
	chdir()
	box := new(sandbox.Sandbox)
	t.Cleanup(box.Cleanup)
	box.Setenv("SECRETS_DIR", "/repo")
	box.Command(secretctl, "init", "tests/keys/master.pub")
	box.Command(secretctl, "cert", "--admin=alice", "--key=tests/keys/alice.pub", "-r", "/users/alice/")
	box.Command(secretctl, "cert", "--admin=alice", "--key=tests/keys/alice.pub", "-rw", "/alice")
	box.Command(secretctl, "cert", "--user=bob", "--key=tests/keys/bob.pub", "-rw", "/")
	box.Command(secretctl, "set", "/hello-world", "HELLO_WORLD", "-x", "10d")
	box.Command(secretctl, "set", "/alice/password", "PA$$W0RD!")
	box.Command(secretctl, "set", "/bob/../alice/password", "PA$$W0RD!")
	err := box.Build()
	if err != nil {
		t.Fatal(err)
	}
	err = box.Mkdir("/repo", 0777)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := sshAgent(box, "tests/keys/master", "tests/keys/alice", "tests/keys/bob")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(agent.Stop)

	result, err := box.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("stderr and/or return code check failed:\n%s", result)
	}
	for _, path := range []string{
		"/repo/secrets/hello-world.x",
		"/repo/access/admin/alice.01.cert",
		"/repo/access/admin/alice.02.cert",
		"/repo/access/master.cert",
	} {
		if !box.Exists(path) {
			t.Errorf("Expected path not found in box: %s", path)
		}
	}
	if testing.Verbose() {
		t.Logf("\n%s", result)
	}
}
