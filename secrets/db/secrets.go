package db

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/ssh"

	"secrets/crypto"
)

type secretPlainText struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Access  string `json:"access"`
	Expires uint   `json:"expires"`
}

func setSecret(ctx context.Context, sql sqlable, signer ssh.Signer, userkey string, query *Query) ([]string, error) {
	// TODO: access control
	// TODO: default expiration
	var secrets []secretPlainText
	err := json.Unmarshal(*query.Items, &secrets)
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	const dbQuery = `
		INSERT INTO secret(namespace, key, value, access, expires)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (namespace, key) DO
		UPDATE SET value = excluded.value,
		           access = excluded.access,
		           expires = excluded.expires
	`
	insert, err := sql.PrepareContext(ctx, dbQuery)
	if err != nil {
		return nil, fmt.Errorf("compiling sql: %w", err)
	}
	defer insert.Close()
	changed := make([]string, 0, len(secrets))
	var v = new(crypto.SecretValue)
	for _, s := range secrets {
		err = v.Encrypt(signer, s.Value, query.Namespace, s.Key, s.Access)
		if err != nil {
			return nil, fmt.Errorf("encrypting secret %s/%s: %w", query.Namespace, s.Key, err)
		}
		_, err = insert.ExecContext(ctx, query.Namespace, s.Key, v, s.Access, s.Expires)
		if err != nil {
			return nil, fmt.Errorf("writing secret %s/%s: %w", query.Namespace, s.Key, err)
		}
		changed = append(changed, s.Key)
	}
	return changed, nil
}
