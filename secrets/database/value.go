package database

import "secrets/crypto"

func (db *DB) encryptValue(path []string, value []byte) (encrypted []byte, err error) {
	return crypto.Encrypt(db.key, path, value)
}

func (db *DB) decryptValue(path []string, encrypted []byte) (value []byte, err error) {
	return crypto.Decrypt(db.key, path, encrypted)
}
