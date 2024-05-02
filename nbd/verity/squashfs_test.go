package verity

import (
	"testing"

	"os"
	"path/filepath"
)

func TestQuickAndDirty(t *testing.T) {
	path := "testdata/pseudorandom.squashfs"
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	stat, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	verity, err := verityAfterSquashfs(file)
	if err != nil {
		t.Fatal(err)
	}
	err = verity.Verify(file, 0, 1)
	if err != nil {
		t.Fatalf("verify first byte: %v", err)
	}
	err = verity.Verify(file, 0, 1<<10)
	if err != nil {
		t.Fatalf("verify more than first block: %v", err)
	}
	err = verity.Verify(file, 100, 1)
	if err != nil {
		t.Fatalf("verify a byte at some offset: %v", err)
	}
	err = verity.Verify(file, 0, int(stat.Size()))
	if err != nil {
		t.Fatalf("verify file size: %v", err)
	}
	err = verity.Verify(file, 0, int(stat.Size())+100000)
	if err != nil {
		t.Fatalf("verify more than file size: %v", err)
	}
}

func BenchmarkVerifySingleBlock(b *testing.B) {
	path := "testdata/pseudorandom.squashfs"
	path, err := filepath.Abs(path)
	if err != nil {
		b.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		b.Fatal(err)
	}
	verity, err := verityAfterSquashfs(file)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		err = verity.Verify(file, 0, 1)
		if err != nil {
			b.Fatalf("verify first byte: %v", err)
		}
	}
}
