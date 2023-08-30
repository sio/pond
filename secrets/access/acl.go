package access

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/util"
)

// Initialize access control list for master key certificate at provided path
func Open(path string) (*ACL, error) {
	cert, err := util.LoadCertificate(path)
	if err != nil {
		return nil, err
	}
	err = ValidateMasterCert(cert.Key, cert)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(schema)
	if err != nil {
		return nil, fmt.Errorf("sql schema: %w", err)
	}
	return &ACL{master: cert, db: db}, nil
}

// Access control list
type ACL struct {
	master *ssh.Certificate
	db     *sql.DB
}

func (acl *ACL) Close() error {
	if acl.db == nil {
		return nil
	}
	return acl.db.Close()
}

// Load user certificates by path.
// All previously known user certificates will be forgotten.
func (acl *ACL) LoadUser(paths []string) error {
	return acl.loadCerts(paths, false)
}

// Load administrator certificates by path.
// All previously known administrator certificates will be forgotten.
func (acl *ACL) LoadAdmin(paths []string) error {
	return acl.loadCerts(paths, true)
}

func (acl *ACL) loadCerts(paths []string, admin bool) error {
	tx, err := acl.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var remove [2]uint8
	if admin {
		remove[0] = caps[ManageReaders]
		remove[1] = caps[ManageWriters]
	} else {
		remove[0] = caps[Read]
		remove[1] = caps[Write]
	}
	_, err = tx.Exec(
		"DELETE FROM ACL WHERE Capability = ? OR Capability = ?",
		remove[0],
		remove[1],
	)
	if err != nil {
		return fmt.Errorf("sql delete: %w", err)
	}
	for _, path := range paths {
		cert, err := util.LoadCertificate(path)
		if err != nil {
			return fmt.Errorf("%w: %s", err, path)
		}
		err = acl.Validate(cert, admin)
		if err != nil {
			return fmt.Errorf("%w: %s", err, path)
		}
		fingerprint := ssh.FingerprintSHA256(cert.Key)
		for _, p := range cert.ValidPrincipals {
			if p[len(p)-1] != '/' {
				p += "/"
			}
			for c := range cert.Permissions.CriticalOptions {
				_, err = tx.Exec(
					"INSERT INTO ACL(Fingerprint, Capability, Path, ValidAfter, ValidBefore) VALUES (?, ?, ?, ?, ?)",
					fingerprint,
					caps[Capability(c)],
					p,
					cert.ValidAfter,
					cert.ValidBefore,
				)
				if err != nil {
					return fmt.Errorf("sql insert: %w: %s: %s [%s]", err, path, p, c)
				}
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// Validate access certificate
func (acl *ACL) Validate(cert *ssh.Certificate, admin bool) error {
	if len(cert.KeyId) == 0 {
		return fmt.Errorf("empty key id")
	}
	if len(cert.ValidPrincipals) == 0 {
		return fmt.Errorf("no allowed paths listed in principals field")
	}
	for _, p := range cert.ValidPrincipals {
		if p[0] != '/' {
			return fmt.Errorf("relative paths not allowed in principals field: %q", p)
		}
	}
	if len(cert.Permissions.CriticalOptions) == 0 {
		return fmt.Errorf("no capabilities listed in critical options field")
	}
	for c := range cert.Permissions.CriticalOptions {
		capability := Capability(c)
		if !capability.Valid() {
			return fmt.Errorf("invalid capability: %s", c)
		}
		if capability.Admin() != admin {
			return fmt.Errorf("mixing user and administrator capabilities in one certificate")
		}
	}
	if admin && !pubEqual(acl.master.Key, cert.SignatureKey) {
		return fmt.Errorf("certificate was not signed by master key: %s", cert.KeyId)
	}
	if !admin {
		for _, p := range cert.ValidPrincipals {
			for c := range cert.Permissions.CriticalOptions {
				err := acl.Check(cert.SignatureKey, Required[Capability(c)], p)
				if err != nil {
					return fmt.Errorf("signer not valid: %w", err)
				}
			}
		}
	}
	var supported = make([]string, 2)
	if admin {
		supported[0] = string(ManageReaders)
		supported[1] = string(ManageWriters)
	} else {
		supported[0] = string(Read)
		supported[1] = string(Write)
	}
	validator := &ssh.CertChecker{
		SupportedCriticalOptions: supported,
	}
	err := validator.CheckCert(cert.ValidPrincipals[0], cert)
	if err != nil {
		return err
	}
	return nil
}

// Check if access is allowed
func (acl *ACL) Check(key ssh.PublicKey, c Capability, dir string) error {
	if dir[len(dir)-1] != '/' {
		dir += "/"
	}
	fingerprint := ssh.FingerprintSHA256(key)
	const query = `
		SELECT count(Capability)
		FROM ValidACL
		WHERE Fingerprint = ? AND Capability = ? AND Path GLOB ? || "*"
	`
	var count int
	err := acl.db.QueryRow(query, fingerprint, caps[c], dir).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrPermissionDenied
	}
	return nil
}

var ErrPermissionDenied = errors.New("permission denied")
