package crypto

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

var (
	magicHeader    = []byte("pond/secret")
	magicSeparator = []byte{0, '\n', '\r', '\n', 0}
)

// Encrypt sensitive value
func Encrypt(signer ssh.Signer, path []string, value []byte) (cipher []byte, err error) {
	return v1encrypt(signer, path, value)
}

// Decrypt sensitive value
func Decrypt(signer ssh.Signer, path []string, cipher []byte) (value []byte, err error) {
	if len(cipher) == 0 {
		return nil, nil
	}
	switch cipher[0] {
	case v1tag:
		return v1decrypt(signer, path, cipher)
	default:
		return nil, fmt.Errorf("unsupported encryption version %d", cipher[0])
	}
}
