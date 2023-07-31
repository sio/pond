package db

import (
	"context"
	"encoding/json"
	"fmt"
)

type usergroup struct {
	User      string `json:"user"`
	Usergroup string `json:"usergroup"`
}

func setUsergroup(ctx context.Context, sql sqlable, payload *json.RawMessage) ([]usergroup, error) {
	var groups []usergroup
	err := json.Unmarshal(*payload, &groups)
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	const query = `
		INSERT INTO usergroup(usergroup, user)
		VALUES (?, ?)
		ON CONFLICT (usergroup, user) DO NOTHING
	`
	insert, err := sql.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("compiling sql: %w", err)
	}
	defer insert.Close()
	for _, g := range groups {
		_, err := insert.ExecContext(ctx, g.Usergroup, g.User)
		if err != nil {
			return nil, fmt.Errorf("writing usergroup=%v: %w", g, err)
		}
	}
	return groups, nil
}
