package crypto

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

var (
	magicHeader    = []byte("pond/secret")
	magicSeparator = []byte{0, '\n', '\r', '\n', 0}
)

// Store sensitive value in encrypted form and decrypt on demand.
// Original unencrypted value is never saved
type SecretValue []byte

// Encrypt sensitive value
func (s *SecretValue) Encrypt(signer ssh.Signer, value string, keywords ...string) error {
	return s.v1encrypt(signer, value, keywords...)
}

// Decrypt sensitive value
func (s *SecretValue) Decrypt(signer ssh.Signer, keywords ...string) (string, error) {
	if len(*s) == 0 {
		return "", nil
	}
	switch (*s)[0] {
	case 1:
		return s.v1decrypt(signer, keywords...)
	default:
		return "", fmt.Errorf("unsupported SecretValue version %d", (*s)[0])
	}
}
