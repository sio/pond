package ldd

import (
	"testing"

	"io"
	"os"
	"path/filepath"
)

func TestReadLdsoCache(t *testing.T) {
	cache, err := ldsoCache("")
	if os.IsNotExist(err) {
		t.Skip(err)
	}
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		artifacts := os.Getenv("ARTIFACTS")
		if artifacts == "" {
			t.Log("Not saving debug files, $ARTIFACTS variable is empty")
			return
		}
		err := copyFile("/etc/ld.so.cache", artifacts)
		if err != nil {
			t.Errorf("Failed to save ld.so.cache for debugging: %v", err)
		} else {
			t.Logf("Saved ld.so.cache for debugging later: %s", artifacts)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	var count uint
	for key, value := range cache {
		if count > 10 {
			t.Log("...")
			break
		}
		t.Logf("%s => %s", key, value)
		count++
	}
	t.Logf("ld.so.cache: %d entries", len(cache))
	if len(cache) == 0 {
		t.FailNow()
	}
}

func copyFile(src, destdir string) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = from.Close() }()

	err = os.MkdirAll(destdir, 0700)
	if err != nil {
		return err
	}

	dest := filepath.Join(destdir, filepath.Base(src))
	to, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = to.Close() }()

	_, err = io.Copy(to, from)
	return err
}
