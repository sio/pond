package tests

import (
	"testing"

	"secrets/crypto"

	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
)

// Use local private key for signing messages in tests
func Signer(keyname string) (crypto.SignerFunc, error) {
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

func TestEncrypt(t *testing.T) {
	sign, err := Signer("keys/storage")
	if err != nil {
		t.Fatal(err)
	}

	var value = "test-secret-value"
	var keywords = []string{"test-namespace", "test-keyword"}

	var secret crypto.SecretValue
	err = secret.Encrypt(sign, value, keywords...)
	if err != nil {
		t.Fatal(err)
	}
	output, err := secret.Decrypt(sign, keywords...)
	if err != nil {
		t.Fatal(err)
	}
	if output != value {
		t.Fatalf("value got mangled during encryption: original=%q, modified=%q", value, output)
	}
	t.Logf("%x", secret)
}
