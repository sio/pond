package tests

import (
	"testing"

	"secrets/crypto"
	"secrets/database"

	"context"
	"os"
	"path/filepath"
	"time"
)

func TestReadWriteDB(t *testing.T) {
	key, err := crypto.LocalKey("keys/storage")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("", "TestReadWriteDB-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	db, err := database.Open(filepath.Join(dir, "test.db"), key)
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	tests := []struct {
		path  []string
		value string
	}{
		{[]string{"short", "path"}, "short value"},
		{[]string{"a", "slightly", "longer", "path"}, "a slightly longer value"},
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	for idx, tt := range tests {
		err := db.Set(ctx, tt.path, []byte(tt.value))
		if err != nil {
			t.Fatalf("writing value #%d: %v", idx, err)
		}
	}

	err = db.Close()
	if err != nil {
		t.Fatalf("closing database: %v", err)
	}
	db, err = database.Open(filepath.Join(dir, "test.db"), key)
	if err != nil {
		t.Fatalf("reopening database: %v", err)
	}
	for idx, tt := range tests {
		value, err := db.Get(ctx, tt.path)
		if err != nil {
			t.Errorf("reading value #%d: %v", idx, err)
			meta, err := db.GetMetadata(ctx, tt.path)
			if err != nil {
				t.Errorf("reading metadata #%d: %v", idx, err)
			} else if meta.Expired() {
				t.Errorf("path #%d expired: %v", idx, tt.path)
			}
			continue
		}
		if string(value) != tt.value {
			t.Fatalf("value has changed:\nwant: %s\n got: %s", tt.value, string(value))
		}
	}
}
