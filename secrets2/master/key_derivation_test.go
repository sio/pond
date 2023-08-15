package master

import (
	"testing"

	"bytes"
	"crypto/rand"
	"golang.org/x/crypto/nacl/box"
)

func TestBoxKeyDerivation(t *testing.T) {
	for i := 0; i < 100; i++ {
		pubKey, privKey, err := box.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		derivedKey := deriveBoxKey(privKey)
		if !bytes.Equal(pubKey[:], derivedKey[:]) {
			t.Fatalf("derived public key mismatch:\n got: %x\nwant: %x", derivedKey, pubKey)
		}
	}
}
