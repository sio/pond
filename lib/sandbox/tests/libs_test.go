//go:build linux

package tests

import (
	"github.com/sio/pond/lib/sandbox"
	"testing"
)

func TestWithLibs(t *testing.T) {
	box := new(sandbox.Sandbox)
	t.Cleanup(box.Cleanup)
	box.CommandWithLibs("sh", "-c", `echo "Shell: $0; PID: $$"`)
	box.CommandWithLibs("cat", "libs_test.go")
	box.CommandWithLibs("du", "-sh", "/")
	box.CommandWithLibs("find", "/")
	result, err := box.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("Exit code: %d\nOutput:\n%s", result.ExitCode(), result.Output())
	}
	if testing.Verbose() {
		t.Logf("Output:\n%s", result.Output())
	}
}
