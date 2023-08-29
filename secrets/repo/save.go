package repo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/access"
)

// Save objects to repository
func (r *Repository) Save(x any) (path string, err error) {
	cert, isCert := x.(*ssh.Certificate)
	switch {
	case isCert:
		return r.saveCert(cert)
	default:
		return "", fmt.Errorf("can not save %T to repository", x)
	}
}

func (r *Repository) saveCert(cert *ssh.Certificate) (path string, err error) {
	if len(cert.ValidPrincipals) == 0 {
		return "", fmt.Errorf("empty valid principals in ssh certificate")
	}
	var isAdmin *bool
	for p := range cert.Permissions.CriticalOptions {
		capability := access.Capability(p)
		switch {
		case capability.User():
			if isAdmin == nil {
				isAdmin = new(bool)
			}
			if *isAdmin {
				return "", fmt.Errorf("mixing user and administrator capabilities in one cert is not supported")
			}
		case capability.Admin():
			if isAdmin == nil {
				isAdmin = new(bool)
				*isAdmin = true
			}
			if !(*isAdmin) {
				return "", fmt.Errorf("mixing user and administrator capabilities in one cert is not supported")
			}
		default:
			return "", fmt.Errorf("invalid capability: %s", capability)
		}
	}
	if *isAdmin {
		path = filepath.Join(r.root, accessDir, adminDir, cert.KeyId+certExt)
	} else {
		path = filepath.Join(r.root, accessDir, usersDir, cert.KeyId+certExt)
	}
	var backup string
	var suffix int = 1
	_, err = os.Stat(path)
	for !errors.Is(err, os.ErrNotExist) {
		if err != nil {
			return "", err
		}
		backup, _ = strings.CutSuffix(path, certExt)
		backup = fmt.Sprintf("%s.x%02X%s", backup, suffix, certExt)
		_, err = os.Stat(backup)
		suffix++
	}
	if backup != "" {
		err = os.Rename(path, backup)
		if err != nil {
			return "", err
		}
	}
	out, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return "", err
	}
	_, err = out.Write(ssh.MarshalAuthorizedKey(cert))
	if err != nil {
		return "", err
	}
	return path, nil
}
