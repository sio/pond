package repo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
		if !capability.Valid() {
			return "", fmt.Errorf("invalid capability: %s", capability)
		}
		if isAdmin == nil {
			isAdmin = new(bool)
			*isAdmin = capability.Admin()
		}
		if *isAdmin != capability.Admin() {
			return "", fmt.Errorf("mixing user and administrator capabilities in one cert is not supported")
		}
	}
	var prefix string
	if *isAdmin {
		prefix = filepath.Join(r.root, accessDir, adminDir, cert.KeyId)
	} else {
		prefix = filepath.Join(r.root, accessDir, usersDir, cert.KeyId)
	}
	const base = 36 // max base supported by FormatInt; gives 1296 sortable two-character indexes
	var suffix int64
	if existing, _ := filepath.Glob(prefix + "*" + certExt); len(existing) > 0 {
		last := existing[len(existing)-1]
		last = strings.TrimPrefix(last, prefix+".")
		last = strings.TrimSuffix(last, certExt)
		suffix, err = strconv.ParseInt(last, base, 64)
		if err != nil {
			suffix = 0
		}
	}
	for {
		suffix++
		path = fmt.Sprintf("%s.%02s%s", prefix, strconv.FormatInt(suffix, base), certExt)
		_, err = os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
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
