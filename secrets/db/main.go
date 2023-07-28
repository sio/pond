package db

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh"
)

type Database struct {
	sql *sql.DB
	key ssh.Signer
}

// Open local file for storing secrets in it. File will be created if missing.
func Open(filename string, key ssh.Signer) (*Database, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	sql, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	err = applyAllMigrations(sql)
	if err != nil {
		return nil, err
	}
	db := &Database{
		sql: sql,
		key: key,
	}
	err = db.verifyEncryptionKey()
	if errors.Is(err, errKeyNotDefined) {
		err = db.setEncryptionKey()
		if err != nil {
			return nil, err
		}
		err = db.verifyEncryptionKey()
	}
	if err != nil {
		return nil, err
	}
	err = db.createInitialAdminAccount()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (db *Database) Close() error {
	var err error
	if key, ok := db.key.(io.Closer); ok {
		err = key.Close()
	}
	return errors.Join(db.sql.Close(), err)
}

func (db *Database) createInitialAdminAccount() error {
	const query = `SELECT count(key) FROM key LEFT JOIN user ON key.user = user.user WHERE user.admin != 0`
	var count uint
	row := db.sql.QueryRow(query)
	err := row.Scan(&count)
	if count > 0 {
		return nil
	}
	if err != nil {
		return fmt.Errorf("counting admin keys: %s", err)
	}

	const insert = `
		INSERT INTO user(user, admin)
		VALUES ("systemUser", true);

		INSERT INTO key(user, key)
		SELECT "systemUser" as user, key
		FROM encryption
		ORDER BY timestamp DESC
		LIMIT 1;
	`
	var salt = make([]byte, 6)
	_, err = rand.Read(salt)
	if err != nil {
		return fmt.Errorf("random: %w", err)
	}
	var username = fmt.Sprintf("system-%x", salt)
	_, err = db.sql.Exec(strings.ReplaceAll(insert, "systemUser", username))
	if err != nil {
		return fmt.Errorf("inserting initial admin key: %w", err)
	}
	log.Printf("IMPORTANT: initial administrator account (%s) uses current encryption key for authentication. You may want to disable it after initial setup", username)
	return nil
}
