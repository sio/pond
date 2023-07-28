package db

import (
	"context"
	"fmt"
)

type user struct {
	User     string
	Admin    bool
	Disabled bool
}

func (db *Database) setUser(ctx context.Context, u *user) error {
	const query = `
		INSERT INTO user(user, admin, disabled)
		VALUES (?, ?, ?)
		ON CONFLICT (user) DO
		UPDATE SET admin=excluded.admin, disabled=disabled.admin
	`
	_, err := db.sql.ExecContext(ctx, query, u.User, u.Admin, u.Disabled)
	if err != nil {
		return fmt.Errorf("insert into user: %w", err)
	}
	return nil
}
