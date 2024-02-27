package ldd

import (
	"testing"

	"github.com/sio/pond/lib/sandbox"
	"os"
	"sort"
	"strings"
)

func TestReference(t *testing.T) {
	progs := []string{
		"/bin/ls",
		"/bin/bash",
		"/bin/sh",
		"/bin/mount",
		"/bin/test",
	}
	for _, exe := range progs {
		t.Run(exe, func(t *testing.T) {
			if _, err := os.Stat(exe); os.IsNotExist(err) {
				t.Skipf("%s does not exist", exe)
			}
			want, err := sandbox.Ldd(exe)
			if err != nil {
				t.Fatalf("reference ldd: %v", err)
			}
			got, err := Depends(exe)
			if err != nil {
				t.Fatalf("our ldd: %v", err)
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
					t.FailNow()
				}
			}
		})
	}
}
