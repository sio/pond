package verity

import (
	"testing"

	"os"
	"path/filepath"
)

func TestQuickAndDirty(t *testing.T) {
	path := "../../rootfs/x/squashfs/ok.squashfs"
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	verity, err := verityAfterSquashfs(file)
	if err != nil {
		t.Fatal(err)
	}
	err = verity.Verify(file, 0) // TODO: this does not work yet, probably error in calculating of leafHashOffset or index-hash-offset
	if err != nil {
		t.Fatal(err)
	}
}
