package database

import (
	"context"
	"database/sql"
	"fmt"
)

type Tx struct {
	tx     *sql.Tx
	db     *DB
	ctx    context.Context
	cancel func()
	unlock func()
}

func (db *DB) BeginTx(ctx context.Context) (*Tx, error) {
	var err error
	err = db.txLock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	t := new(Tx)
	t.db = db
	t.unlock = db.txUnlock
	t.ctx, t.cancel = context.WithCancel(ctx)
	t.tx, err = db.sql.BeginTx(t.ctx, nil)
	if err != nil {
		t.cleanup()
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	return t, nil
}

func (t *Tx) Get(path []string) (value []byte, err error) {
	return t.db.get(t.ctx, t.tx, path)
}

func (t *Tx) Set(path []string, value []byte) (err error) {
	return t.db.set(t.ctx, t.tx, path, value, true)
}

func (t *Tx) Create(path []string, value []byte) (err error) {
	return t.db.set(t.ctx, t.tx, path, value, false)
}

func (t *Tx) GetMetadata(path []string) (meta *Metadata, err error) {
	return t.db.getmeta(t.ctx, t.tx, path)
}

func (t *Tx) Commit() error {
	var err error
	if t.tx != nil {
		err = t.tx.Commit()
	}
	t.cleanup()
	return err
}

func (t *Tx) Rollback() error {
	var err error
	if t.tx != nil {
		err = t.tx.Rollback()
	}
	t.cleanup()
	return err
}

func (t *Tx) cleanup() {
	if t.cancel != nil {
		t.cancel()
	}
	if t.unlock != nil {
		t.unlock()
	}
}
