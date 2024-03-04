package ldd

import (
	"testing"

	"fmt"
	"github.com/sio/pond/lib/sandbox"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	comparisonMaxCount = 100
)

func TestReference(t *testing.T) {
	progs := make(map[string]struct{})
	for _, dir := range []string{"/bin", "/sbin", "/usr/bin", "/usr/sbin"} {
		err := walk(dir, &progs)
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	for exe := range progs { // automatically randomized by Go runtime
		if count > comparisonMaxCount {
			break
		}
		if _, err := os.Stat(exe); os.IsNotExist(err) {
			t.Skipf("%s does not exist", exe)
		}
		want, err := sandbox.Ldd(exe)
		if err != nil {
			if strings.HasSuffix(err.Error(), "=> not found") {
				continue // if reference implementation failed, we won't even try
			}
			t.Errorf("%s: reference ldd: %v", exe, err)
			continue
		}
		if len(want) == 0 {
			continue // testing zero cases against reference implementation is not interesting
		}
		count++
		got, err := Depends(exe)
		if err != nil {
			t.Errorf("%s: our ldd: %v", exe, err)
			continue
		}
		sort.Strings(want)
		sort.Strings(got)
		if len(want) != len(got) {
			t.Errorf("libraries count mismatch: our %d, reference %d", len(got), len(want))
		}
		for i, w := range want {
			if w != got[i] {
				t.Error("output does not match reference")
				t.Logf("reference output:\n%s", strings.Join(want, "\n"))
				t.Logf("actual output:\n%s", strings.Join(got, "\n"))
				break
			}
		}
	}
}

func walk(dir string, exe *map[string]struct{}) error {
	return fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		fullpath := filepath.Join(dir, path)
		stat, err := os.Stat(fullpath)
		if err != nil {
			return fmt.Errorf("%s: %w", fullpath, err)
		}
		if stat.IsDir() {
			return nil
		}
		if stat.Mode()&0b001001001 == 0 { // check executable bit
			return nil
		}
		(*exe)[fullpath] = struct{}{} // add executable path to results
		return nil
	})
}
