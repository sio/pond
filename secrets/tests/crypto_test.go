package tests

// TODO: save some known good data points (plain text -> encrypted)
// TODO: use second implementation of v1 algorithm to verify correctness

import (
	"testing"

	"secrets/crypto"
)

func FuzzEncrypt(f *testing.F) {
	sign, err := crypto.LocalKey("keys/storage")
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
