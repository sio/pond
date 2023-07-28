package db

import (
	"context"
	"encoding/json"
	"fmt"
)

type user struct {
	User     string
	Admin    bool
	Disabled bool
}

func setUser(ctx context.Context, sql sqlable, payload *json.RawMessage) (names []string, err error) {
	var users []user
	err = json.Unmarshal(*payload, &users)
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	const query = `
		INSERT INTO user(user, admin, disabled)
		VALUES (?, ?, ?)
		ON CONFLICT (user) DO
		UPDATE SET admin=excluded.admin, disabled=excluded.disabled
	`
	insert, err := sql.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("compiling sql: %w", err)
	}
	defer insert.Close()
	names = make([]string, 0, len(users))
	for _, u := range users {
		_, err := insert.ExecContext(ctx, u.User, u.Admin, u.Disabled)
		if err != nil {
			return nil, fmt.Errorf("writing user=%q: %w", u.User, err)
		}
		names = append(names, u.User)
	}
	return names, nil
}
