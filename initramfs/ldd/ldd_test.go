package ldd

import (
	"testing"

	"debug/elf"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

func TestLddManually(t *testing.T) {
	target := os.Getenv("LDD_TARGET")
	if target == "" {
		t.Skipf("$LDD_TARGET is not set")
	}
	target, err := exec.LookPath(target)
	if err != nil {
		t.Fatal(err)
	}
	deps, err := Depends(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) == 0 {
		t.Fatal("failed to find dependencies")
	}
	t.Logf("%s => %v", target, deps)
	for _, path := range deps {
		stat, err := os.Stat(path)
		if err != nil {
			t.Error(err)
		}
		if stat.Mode().Type()&(fs.ModeDir|fs.ModeDevice|fs.ModeSocket|fs.ModeNamedPipe|fs.ModeCharDevice) != 0 {
			t.Errorf("unexpected file type: %s (%s)", path, stat.Mode().Type())
		}
	}
}

func TestListInterpreters(t *testing.T) {
	if os.Getenv("LDD_LIST_INTERPRETERS") == "" {
		t.Skipf("$LDD_LIST_INTERPRETERS is not set")
	}
	for _, dir := range []string{"/bin", "/sbin", "/usr/bin", "/usr/sbin"} {
		files, err := filepath.Glob(dir + "/*")
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			exe, err := elf.Open(file)
			if err != nil {
				continue
			}
			interp, err := interpreter(exe)
			_ = exe.Close()
			if err == nil {
				t.Logf("%s => %s", file, interp)
			}
		}
	}
}
