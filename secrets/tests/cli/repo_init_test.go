//go:build clitest
package cli

import (
	"testing"
	"clitest"

	"runtime"
	"fmt"
	"os"
)

var secretctl = fmt.Sprintf("bin/secretctl@%s-%s", runtime.GOOS, runtime.GOARCH)

func TestRepoInitialization(t *testing.T) {
	os.Chdir("../..")
	cli, err := clitest.New(
		[][]string{
			{secretctl, "init", "tests/keys/master.pub"},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cli.Cleanup)
	err = cli.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if cli.ExitCode() != 0 {
		t.Logf("Exit code: %d", cli.ExitCode())
		t.Logf("Stdout:\n%s", string(cli.Stdout()))
		t.Logf("Stderr:\n%s", string(cli.Stderr()))
		t.Logf("Combined:\n%s", string(cli.Output()))
		t.FailNow()
	}
}
