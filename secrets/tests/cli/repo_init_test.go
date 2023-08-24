//go:build test_cli
package cli

import (
	"testing"
	"sandbox"

	"runtime"
	"fmt"
	"os"
)

var secretctl = fmt.Sprintf("bin/secretctl@%s-%s", runtime.GOOS, runtime.GOARCH)

func TestRepoInitialization(t *testing.T) {
	os.Chdir("../..")
	sandbox := sandbox.Sandbox{}
	t.Cleanup(sandbox.Cleanup)
	sandbox.Command(secretctl, "init", "tests/keys/master.pub")
	result, err := sandbox.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode() != 0 {
		t.Logf("Exit code: %d", result.ExitCode())
		t.Logf("Stdout:\n%s", string(result.Stdout()))
		t.Logf("Stderr:\n%s", string(result.Stderr()))
		t.Logf("Combined:\n%s", string(result.Output()))
		t.FailNow()
	}
}
