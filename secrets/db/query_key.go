package db

import (
	"context"
	"fmt"
)

type key struct {
	User string
	Key  string
}

func (db *Database) setKey(ctx context.Context, k *key) error {
	const query = `
		INSERT INTO key(user, key)
		VALUES (?, ?)
	`
	_, err := db.sql.ExecContext(ctx, query, k.User, k.Key)
	if err != nil {
		return fmt.Errorf("insert into key: %w", err)
	}
	return nil
}
