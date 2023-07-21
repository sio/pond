package crypto

import (
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
)

// Use local private key for signing messages in tests
func LocalKey(keyname string) (SignerFunc, error) {
	raw, err := os.ReadFile(keyname)
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParseRawPrivateKey(raw)
	if err != nil {
		return nil, err
	}
	private, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("%T can not be used for signatures: %w", key, err)
	}
	return func(data []byte) (*ssh.Signature, error) {
		return private.Sign(rand.Reader, data)
	}, nil
}
