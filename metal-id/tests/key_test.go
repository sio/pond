package tests

import (
	"testing"

	"metal_id"

	"crypto"
	"crypto/ed25519"
	_ "embed"
	"golang.org/x/crypto/ssh"
)

//go:embed data/ed25519
var rawPrivateKeyEd25519 []byte

//go:embed data/ed25519.pub
var rawPublicKeyEd25519 []byte

// Both crypto.PublicKey and crypto.PrivateKey are expected to implement
// Equal() but for backwards compatibility reasons this fact is not included
// into type definiton
type comparableKey interface {
	Equal(x crypto.PublicKey) bool
}

// Unexported type x/crypto/ssh:ed25519PublicKey implements CryptoPublicKey()
//
// See:
//
//	https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.11.0:ssh/keys.go;l=608;drc=c6a20f9984ce6da2ddf94d411c9ffc473e87d15e
type cryptoKey interface {
	CryptoPublicKey() crypto.PublicKey
}

func TestPrivateKeyMarshalling(t *testing.T) {
	sshOriginalKey, err := ssh.ParseRawPrivateKey(rawPrivateKeyEd25519)
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}
	key, ok := sshOriginalKey.(*ed25519.PrivateKey)
	if !ok {
		t.Fatalf("failed to convert key from %T to ed25519.PrivateKey", sshOriginalKey)
	}
	encoded, err := metal_id.EncodePrivateKey(key)
	if err != nil {
		t.Fatalf("failed to encode private key: %v", err)
	}
	sshReencodedKey, err := ssh.ParseRawPrivateKey(encoded)
	if err != nil {
		t.Fatalf("failed to parse reencoded key: %v", err)
	}
	reenc, ok := sshReencodedKey.(*ed25519.PrivateKey)
	if !ok {
		t.Fatalf("failed to convert key from %T to ed25519.PrivateKey", sshReencodedKey)
	}
	if !key.Public().(comparableKey).Equal(reenc.Public()) {
		t.Logf("\nOriginal key:\n%s\nReencoded key:\n%s", string(rawPrivateKeyEd25519), string(encoded))
		t.Fatalf("original and reencoded keys do not match")
	}
}

func TestPublicKeyMarshalling(t *testing.T) {
	sshOriginalKey, _, _, _, err := ssh.ParseAuthorizedKey(rawPublicKeyEd25519)
	if err != nil {
		t.Fatalf("failed to parse public key: %v", err)
	}
	key := sshOriginalKey.(cryptoKey).CryptoPublicKey()
	encoded, err := metal_id.EncodePublicKey(key)
	if err != nil {
		t.Fatalf("failed to encode public key: %v", err)
	}
	sshReencodedKey, _, _, _, err := ssh.ParseAuthorizedKey(encoded)
	if err != nil {
		t.Fatalf("failed to parse reencoded key: %v", err)
	}
	if !sshReencodedKey.(cryptoKey).CryptoPublicKey().(comparableKey).Equal(key) {
		t.Logf("\nOriginal key:\n%s\nReencoded key:\n%s", string(rawPublicKeyEd25519), string(encoded))
		t.Fatalf("original and reencoded keys do not match")
	}
}
