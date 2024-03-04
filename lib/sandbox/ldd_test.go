//go:build linux

package sandbox

import (
	"testing"
)

func TestLddHappyPath(t *testing.T) {
	const exe = "/bin/systemd"
	libs, err := ldd(exe)
	if err != nil {
		t.Fatalf("%s: %v", exe, err)
	}
	if len(libs) == 0 {
		t.Fatalf("empty library list for %s", exe)
	}
	t.Logf("%d libraries: %s", len(libs), libs)
}

func TestLddStatic(t *testing.T) {
	const exe = "/sbin/ldconfig"
	libs, err := ldd(exe)
	if err != nil {
		t.Fatalf("%s: %v", exe, err)
	}
	if len(libs) != 0 {
		t.Fatalf("unexpected dependencies for %s: %v", exe, libs)
	}
}

func TestLddNotDynamic(t *testing.T) {
	const exe = "/bin/which" // shell script
	libs, err := ldd(exe)
	if err != nil {
		t.Fatalf("%s: %v", exe, err)
	}
	if len(libs) != 0 {
		t.Fatalf("unexpected dependencies for %s: %v", exe, libs)
	}
}
