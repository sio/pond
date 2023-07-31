package db

import (
	"context"
	"encoding/json"
	"fmt"
)

type namespace struct {
	Namespace      string `json:"namespace"`
	DefaultAccess      string `json:"default_access"`
}

func setNamespace(ctx context.Context, sql sqlable, payload *json.RawMessage) (changed []string, err error) {
	var namespaces []namespace
	err = json.Unmarshal(*payload, &namespaces)
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	const query = `
		INSERT INTO namespace(namespace, default_access)
		VALUES (?, ?)
		ON CONFLICT (namespace) DO
		UPDATE SET default_access=excluded.default_access
	`
	insert, err := sql.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("compiling sql: %w", err)
	}
	defer insert.Close()
	changed = make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		_, err := insert.ExecContext(ctx, ns.Namespace, ns.DefaultAccess)
		if err != nil {
			return nil, fmt.Errorf("writing namespace=%q: %w", ns.Namespace, err)
		}
		changed = append(changed, ns.Namespace)
	}
	return changed, nil
}
