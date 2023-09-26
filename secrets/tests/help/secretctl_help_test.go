//go:build docs && linux && amd64

package help

import (
	"testing"

	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sio/pond/lib/block"
)

var secretctl = fmt.Sprintf("bin/secretctl@%s-%s", runtime.GOOS, runtime.GOARCH)

func TestSecretctlHelp(t *testing.T) {
	const doc = "docs/secretctl_usage.md"

	var commands = [][]string{
		{secretctl, "--help"},
		{secretctl, "init", "--help"},
		{secretctl, "cert", "--help"},
		{secretctl, "set", "--help"},
	}
	os.Chdir("../..")
	var template = new(block.Template)
	for _, cmd := range commands {
		output, err := exec.Command(cmd[0], cmd[1:]...).Output()
		if err != nil {
			t.Fatal(err)
		}
		name := strings.Join(cmd, " ")
		template.Set(name, fmt.Sprintf("$ %s\n%s", name, output))
	}
	err := template.Render(doc)
	if err != nil {
		t.Fatal(err)
	}
}
