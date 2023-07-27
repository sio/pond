package db

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"os"

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
		log.Printf("IMPORTANT: setting encryption key to %s", string(ssh.MarshalAuthorizedKey(key.PublicKey())))
		err = db.setEncryptionKey()
		if err != nil {
			return nil, err
		}
		err = db.verifyEncryptionKey()
	}
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
