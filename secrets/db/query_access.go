package db

import (
	"context"
	"encoding/json"
	"fmt"
)

type access struct {
	Access      string `json:"access"`
	Usergroup   string `json:"usergroup"`
	AllowGet    bool   `json:"allow_get,omitempty"`
	AllowSet    bool   `json:"allow_set,omitempty"`
	AllowDelete bool   `json:"allow_delete,omitempty"`
}

func setAccess(ctx context.Context, sql sqlable, payload *json.RawMessage) (changed []access, err error) {
	var acls []access
	err = json.Unmarshal(*payload, &acls)
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	const query = `
		INSERT INTO access(access, usergroup, allow_get, allow_set, allow_delete)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (access, usergroup) DO
		UPDATE SET allow_get=excluded.allow_get, allow_set=excluded.allow_set, allow_delete=excluded.allow_delete
	`
	insert, err := sql.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("compiling sql: %w", err)
	}
	defer insert.Close()
	changed = make([]access, 0, len(acls))
	for _, acl := range acls {
		_, err := insert.ExecContext(ctx, acl.Access, acl.Usergroup, acl.AllowGet, acl.AllowSet, acl.AllowDelete)
		if err != nil {
			return nil, fmt.Errorf("writing ACL='%s/%s': %w", acl.Access, acl.Usergroup, err)
		}
		changed = append(changed, access{Access: acl.Access, Usergroup: acl.Usergroup})
	}
	return changed, nil
}
