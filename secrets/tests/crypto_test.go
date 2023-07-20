package tests

// TODO: save some known good data points (plain text -> encrypted)
// TODO: use second implementation of v1 algorithm to verify correctness

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

func FuzzEncrypt(f *testing.F) {
	sign, err := Signer("keys/storage")
	if err != nil {
		f.Fatal(err)
	}

	f.Add("test-secret-value", "test-namespace", 3)
	f.Add("test-another-value", "test-namespace-new", 1)

	const maxKeywords = 50
	var secret crypto.SecretValue
	f.Fuzz(func(t *testing.T, value string, keyword string, repeat int) {
		if repeat == 0 {
			repeat = 1
		}
		if repeat < 0 {
			repeat *= -1
		}
		repeat = repeat % maxKeywords
		var keywords = make([]string, repeat)
		for i := 0; i < repeat; i++ {
			keywords[i] = keyword
		}
		err := secret.Encrypt(sign, value, keywords...)
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
	})
}
