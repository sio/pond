//go:build linux

package tests

import (
	"sandbox"
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
		t.Fatalf("Exit code: %d\nOutput:\n%s", result.ExitCode(), string(result.Output()))
	}
	if testing.Verbose() {
		t.Logf("Output:\n%s", string(result.Output()))
	}
}
