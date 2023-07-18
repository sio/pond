package db

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Open local file for storing secrets in it. File will be created if missing.
func Open(filename string) (*SecretStorage, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	err = applyAllMigrations(db)
	if err != nil {
		return nil, err
	}
	return &SecretStorage{
		db: db,
	}, nil
}

type SecretStorage struct {
	db *sql.DB
}

func (s *SecretStorage) Close() error {
	if s == nil {
		return nil
	}
	return s.db.Close()
}
