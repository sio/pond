//go:build linux

package sandbox

import (
	"testing"
)

func TestLddInteractive(t *testing.T) {
	const exe = "/bin/systemd"
	libs, err := ldd(exe)
	if err != nil {
		t.Fatal(err)
	}
	if len(libs) == 0 {
		t.Fatalf("empty library list for %s", exe)
	}
	t.Logf("%d libraries: %s", len(libs), libs)
}
