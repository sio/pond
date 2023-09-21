package repo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sio/pond/secrets/access"
	"github.com/sio/pond/secrets/value"
)

// Save objects to repository
func (r *Repository) Save(x any) (path string, err error) {
	switch v := x.(type) {
	case *access.Certificate:
		return r.saveCert(v)
	case *value.Value:
		return r.saveValue(v)
	default:
		return "", fmt.Errorf("can not save %T to repository", x)
	}
}

func (r *Repository) saveValue(v *value.Value) (path string, err error) {
	var buf = new(bytes.Buffer)
	err = v.Serialize(buf)
	if err != nil {
		return "", err
	}
	data := buf.Bytes()
	for _, p := range v.Path {
		path = filepath.Join(r.root, secretsDir, p+ext)
		err = os.WriteFile(path, data, 0600)
		if err != nil {
			return "", err
		}
	}
	return path, nil
}

func (r *Repository) saveCert(cert *access.Certificate) (path string, err error) {
	err = cert.Validate()
	if err != nil {
		return "", err
	}
	var prefix string
	if cert.Admin() {
		prefix = filepath.Join(r.root, accessDir, adminDir, cert.Name())
	} else {
		prefix = filepath.Join(r.root, accessDir, usersDir, cert.Name())
	}
	const base = 36 // max base supported by FormatInt; gives 1296 sortable two-character indexes
	var suffix int64
	if existing, _ := filepath.Glob(prefix + "*" + certExt); len(existing) > 0 {
		sort.Strings(existing)
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
	_, err = out.Write(cert.Marshal())
	if err != nil {
		return "", err
	}
	return path, nil
}
