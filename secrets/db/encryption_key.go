package db

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"golang.org/x/crypto/ssh"

	"secrets/crypto"
	"secrets/util"
)

var errKeyNotDefined = errors.New("encryption key not defined")

func (db *Database) encryptionKey() (ssh.PublicKey, error) {
	query := `SELECT key FROM encryption ORDER BY timestamp DESC LIMIT 1`
	row := db.sql.QueryRow(query)
	var serialized []byte
	err := row.Scan(&serialized)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errKeyNotDefined
	}
	if err != nil {
		return nil, fmt.Errorf("fetching encryption key: %w", err)
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(serialized)
	if err != nil {
		return nil, fmt.Errorf("parsing encryption key: %w", err)
	}
	return key, nil
}

func (db *Database) setEncryptionKey() error {
	const query = `INSERT INTO encryption(key) VALUES (?)`
	var key = util.KeyText(db.key.PublicKey())
	_, err := db.sql.Exec(query, key)
	if err != nil {
		return fmt.Errorf("setting encryption key: %w", err)
	}
	log.Printf("IMPORTANT: setting encryption key to %s", key)
	return nil
}

func (db *Database) verifyEncryptionKey() error {
	correct, err := db.encryptionKey()
	if err != nil {
		return err
	}
	data := make([]byte, 64)
	_, err = rand.Read(data)
	if err != nil {
		return fmt.Errorf("crypto/rand: %w", err)
	}
	sig, err := crypto.Sign(db.key, data)
	if err != nil {
		return fmt.Errorf("ssh.Signer: %w", err)
	}
	if err = correct.Verify(data, sig); err != nil {
		return fmt.Errorf("signature verification: %w", err)
	}
	return nil
}
