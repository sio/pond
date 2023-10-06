package tests

import (
	"github.com/sio/pond/initramfs/cpio"
	"testing"

	"bytes"
	"os"
	"os/exec"
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

	t.Run("verify", func(t *testing.T) {
		cmd := exec.Command("cpio", "-itF", temp.Name())
		if cmd.Err != nil {
			t.Skipf("cpio tool not available: %v", cmd.Err)
		}
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
		cmd.Stdout, cmd.Stderr = stdout, stderr
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		expected := `deep
deep/nested
deep/nested/path
deep/nested/path/cpio.go
header.go
deep/header_test.go
`
		if stdout.String() != expected {
			t.Fatalf("archive listing does not match expected: want\n%s\n\ngot:\n%s", expected, stdout.String())
		}
	})
}
