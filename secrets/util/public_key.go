package util

import (
	"bytes"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// Load ssh public key from file system
func LoadPublicKey(path string) (ssh.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	if err != nil {
		return nil, err
	}
	if cert, ok := key.(*ssh.Certificate); ok {
		return cert.Key, nil
	}
	return key, nil
}

// Load ssh certificate from file system
func LoadCertificate(path string) (*ssh.Certificate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	if err != nil {
		return nil, err
	}
	cert, ok := key.(*ssh.Certificate)
	if !ok {
		return nil, fmt.Errorf("not an ssh-certificate (%s): %s", key.Type(), path)
	}
	return cert, nil
}

// Check if two ssh keys are the same
func EqualSSH(a, b ssh.PublicKey) bool {
	// Checking if a and b point to the same struct (&a == &b) does not speed
	// things up.  If anything, it might even slow comparison down by ~20ns (or
	// may be that was just random jitter, run BenchmarkPubkeyEqual yourself).
	if a.Type() != b.Type() {
		return false
	}
	// Calculating SHA256 fingerprint strings for comparison is twice as slow
	// as comparing raw byte slices, and on top of that prone to hash collisions.
	//
	// Key comments are not part of ssh.PublicKey structs and will not affect
	// Marshal() equality comparison.
	return bytes.Equal(a.Marshal(), b.Marshal())
}
