package kmod

import (
	"testing"

	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	kmodDir        = "/lib/modules"
	kmodCheckShort = 10
	kmodCheckLong  = 100
)

// Check our modinfo result against reference implementation
func TestModinfo(t *testing.T) {
	check := func(module string) int {
		t.Logf("checking %s", module)
		got, err := Info(module)
		if err != nil {
			t.Fatal(err)
		}
		want, err := modinfo(module)
		if err != nil {
			t.Fatal(err)
		}
		if want.Name == "" {
			// Some vendor provided kernel modules (looking at you, Azure) fail
			// to expose metadata to modinfo utility. Our utility produces more
			// info in that case. We are not here to troubleshoot that mess.
			return 0
		}
		if !reflect.DeepEqual(got, want) {
			t.Error("module info does not match the reference implementation")
			t.Logf("got %s", got)
			t.Logf("want %s", want)
		}
		return 1
	}

	kmod := make(map[string]struct{})
	err := fs.WalkDir(os.DirFS(kmodDir), ".", func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			return nil // ignore filesystem traversal errors
		}
		if filepath.Ext(path) != ".ko" {
			return nil
		}
		kmod[filepath.Join(kmodDir, path)] = struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	for module := range kmod {
		count += check(module)
		if count >= kmodCheckShort && testing.Short() {
			break
		}
		if count >= kmodCheckLong {
			break
		}
	}
	if count < kmodCheckShort {
		t.Errorf("could not find enough kernel modules in %s (%d found)", kmodDir, len(kmod))
	}
}

// Reference modinfo implementation: shell out to system binary
func modinfo(path string) (info Modinfo, err error) {

	// Look for executables in paths typically reserved for superuser;
	// modinfo does not require special privileges but is usually placed there
	search := os.Getenv("PATH")
	sep := fmt.Sprintf("%c", os.PathListSeparator)
	extra := strings.Join([]string{"/sbin", "/usr/sbin"}, sep)
	if search == "" {
		search = extra
	} else {
		search = search + sep + extra
	}
	err = os.Setenv("PATH", search)
	if err != nil {
		return info, err
	}

	// Call modinfo
	cmd := exec.Command("modinfo", "-Falias", path)
	raw, err := cmd.Output()
	if err != nil {
		return info, err
	}
	info.Alias = strings.Split(strings.TrimSpace(string(raw)), "\n")
	if empty(info.Alias) {
		info.Alias = nil
	}

	cmd = exec.Command("modinfo", "-Fdepends", path)
	raw, err = cmd.Output()
	if err != nil {
		return info, err
	}
	info.Depends = strings.Split(strings.TrimSpace(string(raw)), ",")
	if empty(info.Depends) {
		info.Depends = nil
	}

	cmd = exec.Command("modinfo", "-Fname", path)
	raw, err = cmd.Output()
	if err != nil {
		return info, err
	}
	info.Name = strings.TrimSpace(string(raw))

	cmd = exec.Command("modinfo", "-Fvermagic", path)
	raw, err = cmd.Output()
	if err != nil {
		return info, err
	}
	info.Kernel = strings.TrimSpace(string(raw))
	if cur := strings.IndexRune(info.Kernel, ' '); cur > 0 {
		info.Kernel = info.Kernel[:cur]
	}
	return info, err
}

func empty(slice []string) bool {
	if len(slice) == 0 {
		return true
	}
	if len(slice) == 1 && slice[0] == "" {
		return true
	}
	return false
}
