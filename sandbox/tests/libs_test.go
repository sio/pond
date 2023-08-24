//go:build linux

package tests

import (
	"sandbox"
	"testing"
)

func TestWithLibs(t *testing.T) {
	box := new(sandbox.Sandbox)
	t.Cleanup(box.Cleanup)
	box.CommandWithLibs("sh", "-c", "echo $$")
	result, err := box.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("Exit code: %d\nOutput:\n%s", result.ExitCode(), string(result.Output()))
	}
	t.Logf("Output:\n%s", string(result.Output()))
}
