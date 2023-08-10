package database

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"golang.org/x/crypto/pbkdf2"

	"secrets/pack"
	"secrets/util"
)

// Path encryption is not reversible.
//
// To be able to rekey the database after key rotation user has to keep a
// journal of all existing paths somewhere outside the database
// (see 'secrets/journal' package in this repo).
//
// PBKDF2 was chosen as a fast enough and secure enough key derivation function.
// Run 'go test ./database -bench=KDF' to compare different algorithms,
// see 'path_algo_test.go' for details.
func (db *DB) securePath(path []string) (secure []byte, err error) {
	const (
		outputSize = sha256.Size
		iter       = 1024
		saltSize   = sha256.Size
	)

	plain, err := pack.Encode(path)
	if err != nil {
		return nil, fmt.Errorf("encoding path to binary: %w", err)
	}
	signature, err := db.key.Sign(util.FailingReader, plain)
	if err != nil {
		return nil, fmt.Errorf("signature: %w", err)
	}
	if len(signature.Blob) < saltSize*2 {
		return nil, errors.New("signature is too short")
	}
	secure = pbkdf2.Key(
		append(plain, signature.Blob[saltSize:]...),
		signature.Blob[:saltSize],
		iter,
		outputSize,
		sha256.New,
	)
	return secure, nil
}
