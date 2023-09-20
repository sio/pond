package tests

import (
	"github.com/sio/pond/secrets/master"
	"github.com/sio/pond/secrets/value"
	"testing"

	"bytes"
	"crypto/rand"
	"io"
	"time"
)

func TestEncryptDecrypt(t *testing.T) {
	v := &value.Value{
		Path:    []string{"encrypted/value"},
		Created: time.Now(),
		Expires: time.Now().Add(10 * time.Hour),
	}
	cert, err := master.LoadCertificate(certPath)
	if err != nil {
		t.Fatalf("LoadCertificate: %v", err)
	}
	const secret = "secret message"
	err = v.Encrypt(cert, []byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if testing.Verbose() {
		t.Logf("Encrypted: %x", v.Blob)
	}
	signer, err := LocalKey(keyPath)
	if err != nil {
		t.Fatalf("LocalKey: %v", err)
	}
	key, err := master.NewKey(signer, cert)
	if err != nil {
		t.Fatalf("NewKey: %v", err)
	}
	decrypted, err := v.Decrypt(key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decrypted) != secret {
		t.Fatalf("data mangled during encryption/decryption: was %q (%db), now %q (%db)", secret, len(secret), string(decrypted), len(decrypted))
	}
	signer, err = LocalKey("../../tests/keys/alice")
	if err != nil {
		t.Fatalf("LocalKey: %v", err)
	}
	err = v.Sign(signer)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	buf := new(bytes.Buffer)
	err = v.Serialize(buf)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if testing.Verbose() {
		t.Logf(buf.String())
	}
}

func BenchmarkEncryptDecrypt(b *testing.B) {
	v := &value.Value{
		Path:    []string{"hello", "world"},
		Created: time.Now(),
		Expires: time.Now().Add(10 * time.Hour),
	}
	cert, err := master.LoadCertificate(certPath)
	if err != nil {
		b.Fatalf("LoadCertificate: %v", err)
	}
	signer, err := LocalKey(keyPath)
	if err != nil {
		b.Fatalf("LocalKey: %v", err)
	}
	key, err := master.NewKey(signer, cert)
	if err != nil {
		b.Fatalf("NewKey: %v", err)
	}
	var secret = make([]byte, 512)
	for i := 0; i < b.N; i++ {
		_, err = io.ReadFull(rand.Reader, secret)
		if err != nil {
			b.Fatalf("rand: %v", err)
		}
		err = v.Encrypt(cert, []byte(secret))
		if err != nil {
			b.Fatalf("Encrypt: %v", err)
		}
		decrypted, err := v.Decrypt(key)
		if err != nil {
			b.Fatalf("Decrypt: %v", err)
		}
		if !bytes.Equal(decrypted, secret) {
			b.Fatalf("data mangled during encryption/decryption:\nwas %x (%db)\nnow %x (%db)", secret, len(secret), decrypted, len(decrypted))
		}
	}
}
