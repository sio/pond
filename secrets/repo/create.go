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
	certPath, err := filepath.Abs(filepath.Join(path, accessDir, masterCert))
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	_, err = file.Write(master.Marshal()) // TODO: store master public key along with certificate? It seems too difficult to extract the key from cert for writing to known_hosts file
	if err != nil {
		return nil, err
	}
	return Open(path)
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
