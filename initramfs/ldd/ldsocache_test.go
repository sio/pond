package ldd

import (
	"testing"

	"os"
)

func TestReadLdsoCache(t *testing.T) {
	cache, err := ldsoCache("")
	if os.IsNotExist(err) {
		t.Skip(err)
	}
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
