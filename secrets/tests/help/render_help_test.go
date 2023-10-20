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

var (
	secretctl = fmt.Sprintf("bin/secretctl@%s-%s", runtime.GOOS, runtime.GOARCH)
	secretd   = fmt.Sprintf("bin/secretd@%s-%s", runtime.GOOS, runtime.GOARCH)
)

func TestRenderHelp(t *testing.T) {
	os.Chdir("../..")
	var err error
	err = render(
		"docs/secretctl_usage.md",
		[][]string{
			{secretctl, "--help"},
			{secretctl, "init", "--help"},
			{secretctl, "cert", "--help"},
			{secretctl, "set", "--help"},
		},
	)
	if err != nil {
		t.Error(err)
	}
	err = render(
		"docs/secretd_usage.md",
		[][]string{
			{secretd, "--help"},
		},
	)
	if err != nil {
		t.Error(err)
	}
}

func render(path string, commands [][]string) error {
	var template = new(block.Template)
	for _, cmd := range commands {
		output, err := exec.Command(cmd[0], cmd[1:]...).Output()
		if err != nil {
			return err
		}
		name := strings.Join(cmd, " ")
		template.Set(name, fmt.Sprintf("$ %s\n%s", name, output))
	}
	err := template.Render(path)
	if err != nil {
		return err
	}
	return nil
}
