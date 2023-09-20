package value

import (
	"testing"

	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"strings"
	"time"
)

func TestValue(t *testing.T) {
	v := &Value{
		Path:    []string{"TestValue", "hello", "world"},
		blob:    []byte(strings.Repeat("some gibberish here", 100)),
		Created: time.Now(),
		Expires: time.Now().Add(10 * time.Hour),
	}
	err := v.Verify()
	if err == nil {
		t.Fatal("verification passed for a value without any signature")
	}

	// Sign this value
	for name, key := range keys() {
		t.Run(name, func(t *testing.T) {
			err = v.Sign(key)
			if err != nil {
				t.Fatalf("failed to sign value: %v", err)
			}
			err = v.Verify()
			if err != nil {
				t.Fatalf("signature verification: %v", err)
			}

			// Serialize
			var buf = new(bytes.Buffer)
			err = v.Serialize(buf)
			if err != nil {
				t.Fatalf("serialize: %v", err)
			}
			serialized := buf.String()
			if testing.Verbose() {
				t.Logf("\n%s", serialized)
			}
			var v2 = new(Value)
			err = v2.Deserialize(bytes.NewBufferString(serialized))
			if err != nil {
				t.Log(v)
				t.Log(v2)
				t.Fatalf("deserialize: %v", err)
			}
			buf = new(bytes.Buffer)
			err = v2.Serialize(buf)
			if err != nil {
				t.Fatalf("serialize v2: %v", err)
			}
			serializedV2 := buf.String()
			if serialized != serializedV2 {
				t.Logf("\n%s", serializedV2)
				t.Fatal("second serialization produced a different output")
			}
		})
	}
}

func BenchmarkSerialize(b *testing.B) {
	v := &Value{
		Path:    []string{"BenchmarkSerialize", "hello", "world"},
		blob:    []byte(strings.Repeat("some gibberish here", 100)),
		Created: time.Now(),
		Expires: time.Now().Add(10 * time.Hour),
	}
	err := v.Verify()
	if err == nil {
		b.Fatal("verification passed for a value without any signature")
	}

	// Sign this value
	for name, key := range keys() {
		b.Run(name, func(b *testing.B) {
			err = v.Sign(key)
			if err != nil {
				b.Fatalf("failed to sign value: %v", err)
			}
			err = v.Verify()
			if err != nil {
				b.Fatalf("signature verification: %v", err)
			}

			// Serialize
			for i := 0; i < b.N; i++ {
				err = v.Serialize(io.Discard)
				if err != nil {
					b.Fatalf("serialize: %v", err)
				}
			}
		})
	}
}

func BenchmarkSignVerify(b *testing.B) {
	v := &Value{
		Path:    []string{"BenchmarkSignVerify", "hello", "world"},
		blob:    []byte(strings.Repeat("some gibberish here", 100)),
		Created: time.Now(),
		Expires: time.Now().Add(10 * time.Hour),
	}
	err := v.Verify()
	if err == nil {
		b.Fatal("verification passed for a value without any signature")
	}

	// Sign this value
	for name, key := range keys() {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err = v.Sign(key)
				if err != nil {
					b.Fatalf("failed to sign value: %v", err)
				}
				err = v.Verify()
				if err != nil {
					b.Fatalf("signature verification: %v", err)
				}
			}
		})
	}
}

func keys() map[string]ssh.Signer {
	var err error
	keys := make(map[string]crypto.Signer)

	_, keys["ed25519"], err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic("ed25519: " + err.Error())
	}

	keys["ecdsa"], err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic("ecdsa: " + err.Error())
	}

	keys["rsa2048"], err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("rsa2048: " + err.Error())
	}

	result := make(map[string]ssh.Signer)
	for name, key := range keys {
		var sshkey ssh.Signer
		sshkey, err = ssh.NewSignerFromSigner(key)
		if err != nil {
			panic("converting to ssh key: " + err.Error())
		}
		if sshkey.PublicKey().Type() == ssh.KeyAlgoRSA {
			sshkey = &rsaModernSigner{sshkey}
		}
		result[name] = sshkey
	}

	return result
}

type rsaModernSigner struct {
	rsa ssh.Signer
}

func (r *rsaModernSigner) PublicKey() ssh.PublicKey {
	return r.rsa.PublicKey()
}
func (r *rsaModernSigner) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	signer, ok := r.rsa.(ssh.AlgorithmSigner)
	if !ok {
		return nil, fmt.Errorf("custom signature algorithms not supported: %s", r.rsa.PublicKey().Type())
	}
	return signer.SignWithAlgorithm(rand, data, ssh.KeyAlgoRSASHA512)
}
