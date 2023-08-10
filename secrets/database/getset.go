package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	selectQuery = `
		SELECT value
		FROM data
		WHERE path=? AND expires < unixepoch()
	`
	upsertQuery = `
		INSERT INTO data(path, value)
		VALUES (?, ?)
		ON CONFLICT (path) DO
		UPDATE SET value=excluded.value
	`
	insertQuery = `
		INSERT INTO data(path, value)
		VALUES (?, ?)
	`
	metadataQuery = `
		SELECT ctime, mtime, expires
		FROM data
		WHERE path=?
	`
)

type sqli interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (db *DB) get(ctx context.Context, engine sqli, path []string) (value []byte, err error) {
	var cipherPath []byte
	cipherPath, err = db.encryptPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt path: %w", err)
	}

	var cipherValue []byte
	err = engine.QueryRowContext(ctx, selectQuery, cipherPath).Scan(&cipherValue)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sql select: %w", err)
	}

	value, err = db.decryptValue(path, cipherValue)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt value: %w", err)
	}
	return value, nil
}

func (db *DB) set(ctx context.Context, engine sqli, path []string, value []byte, overwrite bool) (err error) {
	var cipherPath, cipherValue []byte
	cipherPath, err = db.encryptPath(path)
	if err != nil {
		return fmt.Errorf("failed to encrypt path: %w", err)
	}
	cipherValue, err = db.encryptValue(path, value)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}
	var query = upsertQuery
	if !overwrite {
		query = insertQuery
	}
	_, err = engine.ExecContext(ctx, query, cipherPath, cipherValue)
	if err != nil {
		return fmt.Errorf("sql insert: %w", err)
	}
	return nil
}

func (db *DB) getmeta(ctx context.Context, engine sqli, path []string) (meta *Metadata, err error) {
	var cipherPath []byte
	cipherPath, err = db.encryptPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt path: %w", err)
	}

	var ctime, mtime, expires int64
	err = engine.QueryRowContext(ctx, metadataQuery, cipherPath).Scan(&ctime, &mtime, &expires)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sql select: %w", err)
	}

	return &Metadata{
		Created:  time.Unix(ctime, 0),
		Modified: time.Unix(mtime, 0),
		Expires:  time.Unix(expires, 0),
	}, nil
}
