package db

import (
	"context"
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
	return nil, nil
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
