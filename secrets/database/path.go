package database

// Even though we do not need to decrypt paths during normal database
// operation, a hash function (or a signature) would not be enough.
//
// Since we want all database lookups to require an encryption key, we also
// must support key rotation. If we were to use only cryptographic signatures
// for path storage (hash + asymmetric encryption), we would not be able to
// re-sign the same path with a new key without fetching that path from
// somewhere else first.
//
// This package is intended to be useful on its own (i.e. without 'journal'
// package from the same repo), hence we must store paths as encrypted values
// instead of simple signatures/hashes.
func (db *DB) encryptPath(path []string) (encrypted []byte, err error) {
	return nil, nil
}

func (db *DB) decryptPath(encrypted []byte) (path []string, err error) {
	return nil, nil
}
