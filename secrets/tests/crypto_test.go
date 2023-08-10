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
	f.Add("0", "0", 99)
	f.Add("", "0", -41)

	const maxKeywords = 50
	var cipher []byte
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
		cipher, err = crypto.Encrypt(sign, keywords, []byte(value))
		if err != nil {
			t.Fatal(err)
		}
		output, err := crypto.Decrypt(sign, keywords, cipher)
		if err != nil {
			t.Fatal(err)
		}
		if string(output) != value {
			t.Fatalf("value got mangled during encryption: original=%q, modified=%q", value, string(output))
		}
	})
}
