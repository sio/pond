package repo

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sio/pond/secrets/value"
)

var errNotFound = errors.New("not found")

// Search for a secret among the list of allowed paths
func (r *Repository) Search(what string, where []string) (*value.Value, error) {
	if len(what) == 0 || len(where) == 0 {
		return nil, errNotFound
	}
	what += ext
	var foundLevel uint
	var found string
outer:
	for _, dir := range where {
		dir = filepath.Clean(dir)
		tail := "initial value"
		var level uint
		for len(tail) > 0 && (found == "" || level < foundLevel) {
			path := filepath.Join(r.root, secretsDir, dir, what)
			_, err := os.Stat(path)
			if err == nil {
				found = path
				foundLevel = level
				if level == 0 { // no need to search further if we have an exact match
					break outer
				}
			}
			dir, tail = filepath.Split(dir)
			level++
		}
	}
	if found != "" {
		return value.Load(found)
	}
	return nil, errNotFound
}
