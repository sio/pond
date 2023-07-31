package db

import (
	"context"
	"encoding/json"
	"fmt"
)

type key struct {
	User string
	Key  string
}

func setKey(ctx context.Context, sql sqlable, payload *json.RawMessage) ([]string, error) {
	var keys []key
	err := json.Unmarshal(*payload, &keys)
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	const query = `
		INSERT INTO key(user, key)
		VALUES (?, ?)
	`
	insert, err := sql.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("compiling sql: %w", err)
	}
	defer insert.Close()
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		_, err := insert.ExecContext(ctx, k.User, k.Key)
		if err != nil {
			return nil, fmt.Errorf("writing key=%q: %w", k.Key, err)
		}
		out = append(out, k.Key)
	}
	return out, nil
}
