//go:build test_cli

package cli

import (
	"github.com/sio/pond/lib/sandbox"
	"testing"

	"strings"
)

func TestSecretctlSetExpired(t *testing.T) {
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
	err = box.Copy("tests/keys/alice.expired-admin.cert", "/repo/access/admin/alice-expired.cert")
	if err != nil {
		t.Fatalf("copy: %v", err)
	}

	// Try to use secretctl after adding expired certificate
	result, err = box.Run(secretctl, "set", "/another", "value")
	if err != nil {
		t.Fatal(err)
	}
	if testing.Verbose() {
		t.Logf("exit code %d\n%s", result.ExitCode(), result)
	}
	if result.ExitCode() != 0 {
		t.Fatalf("after placing expired cert into repo:\n%s", result)
	}
	if !strings.Contains(result.Stderr(), "alice-expired.cert") {
		t.Fatalf("no warning about expired cert on stderr:\n%s", result.Stderr())
	}
}
