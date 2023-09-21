package access

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/agent"
	"github.com/sio/pond/secrets/master"
	"github.com/sio/pond/secrets/util"
)

// Initialize access control list for master key certificate at provided path
func Open(path string) (*ACL, error) {
	cert, err := master.LoadCertificate(path)
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
	master *master.Certificate
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
		cert, err := LoadCertificate(path)
		if err != nil {
			return fmt.Errorf("%w: %s", err, path)
		}
		err = acl.Validate(cert)
		if err != nil {
			return fmt.Errorf("%w: %s", err, path)
		}
		fingerprint := ssh.FingerprintSHA256(cert.PublicKey())
		for _, p := range cert.Paths() {
			if p[len(p)-1] != '/' {
				p += "/"
			}
			for _, c := range cert.Capabilities() {
				_, err = tx.Exec(
					"INSERT INTO ACL(Fingerprint, Capability, Path, ValidAfter, ValidBefore) VALUES (?, ?, ?, ?, ?)",
					fingerprint,
					caps[c],
					p,
					cert.ValidAfter(),
					cert.ValidBefore(),
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
func (acl *ACL) Validate(cert *Certificate) error {
	admin := cert.Admin()
	if admin && !util.EqualSSH(acl.master.PublicKey(), cert.SignatureKey()) {
		return fmt.Errorf("certificate was not signed by master key: %s", cert.Name())
	}
	if !admin {
		for _, p := range cert.Paths() {
			for _, c := range cert.Capabilities() {
				err := acl.Check(cert.SignatureKey(), Required[c], p)
				if err != nil {
					return fmt.Errorf("certificate was not signed by a valid administrator: %w", err)
				}
			}
		}
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

// Connect to ssh-agent and find an identity that has sufficient permissions
func (acl *ACL) FindAgent(caps []Capability, paths []string) (*agent.Conn, error) {
	signer, err := agent.New(nil)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*agent.Conn, error) {
		_ = signer.Close()
		return nil, err
	}
	identities := signer.ListKeys()
	if len(identities) == 0 {
		return fail(fmt.Errorf("no identities available in ssh-agent"))
	}
loop_id:
	for _, id := range identities {
		for _, capability := range caps {
			for _, path := range paths {
				err = acl.Check(id, Required[capability], path)
				if err != nil {
					continue loop_id
				}
			}
		}
		err = signer.SetIdentity(id)
		if err != nil {
			return fail(err)
		}
		return signer, nil
	}
	return fail(fmt.Errorf("ssh-agent: no matching identity out of %d tried", len(identities)))
}
