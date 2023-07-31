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
		response.Errorf("permission denied")
		return response, fmt.Errorf("administrative access denied: %s", pubkey)
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

// Common subset of sql.DB and sql.Tx methods used in this package
type sqlable interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
