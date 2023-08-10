package database

import (
	"context"
	"errors"
)

func (db *DB) txLock(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case _, ok := <-db.txlock:
		if ok {
			return nil
		}
		return errors.New("transactions disabled")
	}
}

func (db *DB) txUnlock() {
	select {
	case db.txlock <- struct{}{}:
	default:
		// Do not block if channel is not ready to receive our message
	}
}
