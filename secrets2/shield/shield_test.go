package shield

import (
	"testing"

	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/ssh"
	"time"
)

func TestShield(t *testing.T) {
	const input = "hello world!"

	// Initialization
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	data := []byte(input)
	shield, err := New(signer, data)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if string(data) == input {
		t.Fatalf("data slice not cleared after shielding")
	}

	// Reading shielded value
	value, err := shield.Value()
	if err != nil {
		t.Fatalf("Value(): %v", err)
	}
	bytes := value.Bytes()
	if string(bytes) != input {
		t.Fatalf("data mangled after shielding: got %q, want %q", string(bytes), input)
	}
	value.Close()
	if string(bytes) == input {
		t.Fatalf("data not cleaned up")
	}

	// Count based prekey expiration
	oldCipher := fmt.Sprintf("%x", shield.cipher)
	var i uint32
	for i = 0; i < prekeyMaxReads*110/100; i++ {
		_, err = shield.Value()
		if err != nil {
			t.Fatalf("Value() loop: %v", err)
		}
	}
	newCipher := fmt.Sprintf("%x", shield.cipher)
	if newCipher == oldCipher {
		t.Fatalf("ciphertext did not change after %d reads: count=%d cipher=%s", i, shield.prekeyReads.Load(), newCipher)
	}

	// Time based prekey expiration
	shield.prekeyExpires = time.Now().Add(-time.Second)
	oldCipher = newCipher
	_, err = shield.Value()
	if err != nil {
		t.Fatalf("Value() read after expiration: %v", err)
	}
	time.Sleep(time.Second / 10) // wait for reshield goroutine to complete
	newCipher = fmt.Sprintf("%x", shield.cipher)
	if newCipher == oldCipher {
		t.Fatalf("ciphertext did not change after expiration: %s", newCipher)
	}
}
