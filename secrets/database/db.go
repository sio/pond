package database

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh"
)

type DB struct {
	key    ssh.Signer
	sql    *sql.DB
	txlock chan struct{}
}

var (
	ErrNotFound   = errors.New("not found")
	ErrPathExists = errors.New("path already exists")
)

func (db *DB) Get(ctx context.Context, path []string) (value []byte, err error) {
	return db.get(ctx, db.sql, path)
}

func (db *DB) Set(ctx context.Context, path []string, value []byte) (err error) {
	return db.set(ctx, db.sql, path, value, true)
}

func (db *DB) Create(ctx context.Context, path []string, value []byte) (err error) {
	return db.set(ctx, db.sql, path, value, false)
}

func (db *DB) GetMetadata(ctx context.Context, path []string) (meta *Metadata, err error) {
	return db.getmeta(ctx, db.sql, path)
}

func Open(filename string, key ssh.Signer) (*DB, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	sql, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	const maxConcurrentTransactions = 1 // because sqlite
	db := &DB{
		key:    key,
		sql:    sql,
		txlock: make(chan struct{}, maxConcurrentTransactions),
	}
	for i := 0; i < maxConcurrentTransactions; i++ {
		db.txUnlock()
	}
	return db, nil
}

func (db *DB) Close() error {
	var err error
	if key, ok := db.key.(io.Closer); ok {
		err = key.Close()
	}
	return errors.Join(db.sql.Close(), err)
}
