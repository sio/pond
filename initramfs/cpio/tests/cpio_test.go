package tests

import (
	"github.com/sio/pond/initramfs/cpio"
	"testing"

	"os"
)

func TestCreateCpio(t *testing.T) {
	tree := []struct{ src, dest string }{
		{"../cpio.go", "deep/nested/path/cpio.go"},
		{"../header.go", "header.go"},
		{"../header_test.go", "deep/header_test.go"},
	}
	temp, err := os.CreateTemp("", "cpio-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := os.Remove(temp.Name())
		if err != nil {
			t.Error(err)
		}
	})
	archive := cpio.New(temp)
	for _, item := range tree {
		err = archive.Copy(item.src, item.dest)
		if err != nil {
			t.Fatalf("cpio: writing %s: %v", item.dest, err)
		}
	}
	err = archive.Close()
	if err != nil {
		t.Fatalf("cpio: close: %v", err)
	}
	err = temp.Close()
	if err != nil {
		t.Fatalf("temp: close: %v", err)
	}
	t.Logf("archive created: %s", temp.Name())
}
