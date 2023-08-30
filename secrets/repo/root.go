// Manipulate secrets repository on local filesystem
package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	accessDir  = "access"
	secretsDir = "secrets"
	usersDir   = "user"
	adminDir   = "admin"
	masterCert = "master.cert"
	ext        = ".x"
	certExt    = ".cert"
)

// Secrets repository on local filesystem
type Repository struct {
	root string
}

func (r *Repository) String() string {
	if len(r.root) == 0 {
		return "<uninitialized>"
	}
	return r.root
}

// Open existing repository from any of its subdirectories
func Open(path string) (*Repository, error) {
	root, err := findRoot(path)
	if err != nil {
		return nil, err
	}
	return &Repository{root}, nil
}

// Find root of secrets repository
func findRoot(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	orig := path
loop:
	for ; ; path = filepath.Dir(path) {
		if len(path) == 0 {
			return "", fmt.Errorf("empty path after transformation")
		}
		if path[len(path)-1] == filepath.Separator { // reached root directory
			return "", fmt.Errorf("not a secrets repository: %s", orig)
		}
		for _, dir := range []string{accessDir, secretsDir} {
			if stat, err := os.Stat(filepath.Join(path, dir)); err != nil || !stat.IsDir() {
				continue loop
			}
		}
		if stat, err := os.Stat(filepath.Join(path, accessDir, masterCert)); err != nil || stat.IsDir() {
			continue loop
		}
		return path, nil
	}
}

// Path to repository master key certificate
func (r *Repository) MasterCert() string {
	if r.root == "" {
		return ""
	}
	return filepath.Join(r.root, accessDir, masterCert)
}

// Paths to user certificates
func (r *Repository) UserCerts() []string {
	return r.listCerts(usersDir)
}

// Paths to administrator certificates
func (r *Repository) AdminCerts() []string {
	return r.listCerts(adminDir)
}

func (r *Repository) listCerts(subdir string) []string {
	if r.root == "" {
		return nil
	}
	directory := filepath.Join(r.root, accessDir, subdir)
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	var paths []string
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), certExt) {
			continue
		}
		paths = append(paths, filepath.Join(directory, file.Name()))
	}
	return paths
}
