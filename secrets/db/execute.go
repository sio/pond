package db

import (
	"context"
)

func (db *Database) Execute(ctx context.Context, pubkey string, query *Query) (*Response, error) {
	return nil, nil
}

func (db *Database) ExecuteAdmin(ctx context.Context, pubkey string, query *Query) (*Response, error) {
	return nil, nil
}
