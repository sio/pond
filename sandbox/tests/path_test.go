//go:build linux

package tests

import (
	"sandbox"
	"testing"

	"strings"
)

func TestPathTraversal(t *testing.T) {
	box := new(sandbox.Sandbox)
	t.Cleanup(box.Cleanup)
	box.CommandWithLibs("cat", "../README.md")
	box.CommandWithLibs("find", "/")
	result, err := box.Execute()
	if err == nil {
		output := string(result.Output())
		t.Logf("Successfully executed command that attempts to traverse sandbox path:%s", output)
		t.Logf("Exit code: %d", result.ExitCode())
		t.FailNow()
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("unexpected Sandbox.Execute() error: %v", err)
	}
	if testing.Verbose() {
		t.Log(err)
	}
}
