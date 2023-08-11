package database

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"golang.org/x/crypto/hkdf"
	"io"

	"secrets/pack"
	"secrets/util"
)

// Path encryption is not reversible.
//
// To be able to rekey the database after key rotation user has to keep a
// journal of all existing paths somewhere outside the database
// (see 'secrets/journal' package in this repo).
//
// HKDF was chosen as a fast enough and secure enough key derivation function
// since input key material (ssh signature) is rather long and hard to bruteforce.
// Run 'go test ./database -bench=KDF' to compare different algorithms,
// see 'path_algo_test.go' for details.
func (db *DB) securePath(path []string) (secure []byte, err error) {
	const (
		outputSize = sha256.Size
		iter       = 1024
	)

	plain, err := pack.Encode(path)
	if err != nil {
		return nil, fmt.Errorf("encoding path to binary: %w", err)
	}
	signature, err := db.key.Sign(util.FailingReader, plain)
	if err != nil {
		return nil, fmt.Errorf("signature: %w", err)
	}
	if len(signature.Blob) < outputSize*2 {
		return nil, errors.New("signature is too short")
	}
	salt := sha256.Sum256(signature.Blob)
	kdf := hkdf.New(
		sha256.New,
		append(plain, signature.Blob...),
		salt[:],
		[]byte("pond/secrets: secure path"),
	)
	secure = make([]byte, outputSize)
	_, err = io.ReadFull(kdf, secure)
	if err != nil {
		return nil, fmt.Errorf("reading from HKDF: %w", err)
	}
	return secure, nil
}
