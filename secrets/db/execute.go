package db

import (
	"context"

	"golang.org/x/crypto/ssh"
)

func (db *Database) Execute(ctx context.Context, signer ssh.Signer, pubkey string, query *Query) (*Response, error) {
	return nil, nil
}

func (db *Database) ExecuteAdmin(ctx context.Context, signer ssh.Signer, pubkey string, query *Query) (*Response, error) {
	return nil, nil
}
