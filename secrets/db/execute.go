package db

import (
	"context"
	"database/sql"
	"fmt"
)

func (db *Database) Execute(ctx context.Context, pubkey string, query *Query) (*Response, error) {
	return nil, nil
}

func (db *Database) ExecuteAdmin(ctx context.Context, pubkey string, query *Query) (*Response, error) {
	var response = new(Response)
	if !db.isAdmin(pubkey) {
		response.Errorf("access denied")
		return response, fmt.Errorf("access denied: %s", pubkey)
	}
	var err error
	var tx *sql.Tx
	var sql sqlable
	sql = db.sql
	if query.Action == Set || query.Action == Delete {
		tx, err = db.sql.Begin()
		if err != nil {
			response.Errorf("database error")
			return response, fmt.Errorf("begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback() }()
		sql = tx
	}
	target := fmt.Sprintf("%s/%s", query.Action, query.Namespace)
	switch target {
	case "set/access":
		var items []access
		items, err = setAccess(ctx, sql, query.Items)
		for _, item := range items {
			response.Send(item)
		}
	case "set/key":
		var items []string
		items, err = setKey(ctx, sql, query.Items)
		for _, item := range items {
			response.Send(item)
		}
	case "set/namespace":
		var items []string
		items, err = setNamespace(ctx, sql, query.Items)
		for _, item := range items {
			response.Send(item)
		}
	case "set/user":
		var items []string
		items, err = setUser(ctx, sql, query.Items)
		for _, item := range items {
			response.Send(item)
		}
	case "set/usergroup":
		var items []usergroup
		items, err = setUsergroup(ctx, sql, query.Items)
		for _, item := range items {
			response.Send(item)
		}
	default:
		response.Errorf("not implemented: %s", target)
		return response, response.LastError()
		fmt.Println(sql)
	}
	if err != nil {
		response.Errorf("%s: error. More information in logs", target)
		return response, fmt.Errorf("%s: %w", target, err)
	}
	if tx != nil {
		err = tx.Commit()
		if err != nil {
			response.Errorf("database error")
			return response, fmt.Errorf("commit transaction: %w", err)
		}
	}
	return response, nil
}

func (db *Database) isAdmin(pubkey string) bool {
	const query = `SELECT admin FROM key LEFT JOIN user ON key.user = user.user WHERE key = ?`
	row := db.sql.QueryRow(query, pubkey)
	var result bool
	err := row.Scan(&result)
	if err != nil {
		return false
	}
	return result
}

// Check if a client is allowed to interact with our API
//
// The goal is to drop unauthorized connections (bots, skiddies, random
// lurkers) as early as possible as cheap as possible.
// Role based access control is evaluated much later, when we receive full API
// query and pass it to API endpoint.
//
// This is a very crude check: we compare client's public key against the list
// of all known keys and deny access if we find no matching records.
func (db *Database) AllowAPI(pubkey string) error {
	const query = `SELECT count(key) > 0 FROM key WHERE key = ?`
	row := db.sql.QueryRow(query, pubkey)
	var known bool
	err := row.Scan(&known)
	if err != nil {
		return err
	}
	if !known {
		return fmt.Errorf("not a known key: %s", pubkey)
	}
	return nil
}

// Common subset of sql.DB and sql.Tx methods used in this package
type sqlable interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
