package ldd

import (
	"testing"
)

func TestReadLdsoCache(t *testing.T) {
	cache, err := ldsoCache("")
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range cache {
		t.Logf("%s => %s", key, value)
	}
	t.Logf("ld.so.cache: %d entries", len(cache))
	if len(cache) == 0 {
		t.FailNow()
	}
}
