package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sio/pond/secrets/master"
)

// Create new repository in an empty directory
func Create(path string, master *master.Certificate) (*Repository, error) {
	err := checkRepoEmpty(path)
	if err != nil {
		return nil, err
	}
	for _, subdir := range []string{
		accessDir,
		secretsDir,
		filepath.Join(accessDir, usersDir),
		filepath.Join(accessDir, adminDir),
	} {
		err = os.Mkdir(filepath.Join(path, subdir), 0700)
		if err != nil {
			return nil, err
		}
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	r := &Repository{
		root: path,
	}
	_, err = r.Save(master)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func checkRepoEmpty(path string) error {
	items, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Name() == ".git" {
			continue
		}
		if strings.HasSuffix(item.Name(), ".md") {
			continue
		}
		dir, err := filepath.Abs(path)
		if err != nil {
			dir = path
		}
		return fmt.Errorf("directory not empty: %s", dir)
	}
	return nil
}
