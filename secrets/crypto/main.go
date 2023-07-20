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
func (s *SecretValue) Encrypt(sign SignerFunc, value string, keywords ...string) error {
	return s.v1encrypt(sign, value, keywords...)
}

// Decrypt sensitive value
func (s *SecretValue) Decrypt(sign SignerFunc, keywords ...string) (string, error) {
	if len(*s) == 0 {
		return "", nil
	}
	switch (*s)[0] {
	case 1:
		return s.v1decrypt(sign, keywords...)
	default:
		return "", fmt.Errorf("unsupported secret version %d", (*s)[0])
	}
}

// Helpful alias to frequently used function type
type SignerFunc = func(m []byte) (*ssh.Signature, error)
