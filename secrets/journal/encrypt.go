package journal

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	mathrand "math/rand"

	"golang.org/x/crypto/nacl/secretbox"
)

func (j *Journal) encrypt(plain []byte) (cipher []byte, err error) {
	return j.v1Encrypt(plain)
}

func (j *Journal) decrypt(cipher []byte) (plain []byte, err error) {
	switch j.version {
	case v1:
		return j.v1Decrypt(cipher)
	default:
		return nil, fmt.Errorf("unsupported journal version: %s", j.version)
	}
}

func (j *Journal) v1Encrypt(plain []byte) (cipher []byte, err error) {
	cursor := v1State(j.state)
	if err = cursor.Validate(); err != nil {
		return nil, err
	}

	// Add padding to hide plaintext length
	const (
		minMessageBytes = 64
		maxPaddingBytes = 255 // maximum value a single byte can hold
	)
	var paddingBytes int
	if len(plain) < minMessageBytes {
		paddingBytes += minMessageBytes - len(plain)
	}
	paddingBytes += mathrand.Intn(maxPaddingBytes - paddingBytes)
	if paddingBytes == 0 {
		paddingBytes = 1
	}
	var padding = make([]byte, paddingBytes)
	_, err = io.ReadFull(rand.Reader, padding)
	if err != nil {
		return nil, fmt.Errorf("rand: %w", err)
	}
	padding[0] = byte(paddingBytes)

	// Encrypt using XSalsa20 and Poly1305
	message := append(padding, plain...)
	cipher = secretbox.Seal(nil, message, cursor.Nonce(), cursor.Key())

	// Check for separator collisions
	var chunk []byte
	chunk = append(j.separator, cipher...)
	chunk = append(chunk, j.separator...)
	if bytes.Contains(chunk[1:len(chunk)-1], j.separator) {
		// Retry with a different padding
		return j.v1Encrypt(plain)
	}

	// Update journal state
	nextNonce := sha256.Sum256(message)
	cursor.SetNonce(nextNonce[:])

	return cipher, nil
}

func (j *Journal) v1Decrypt(cipher []byte) (plain []byte, err error) {
	cursor := v1State(j.state)
	if err = cursor.Validate(); err != nil {
		return nil, err
	}
	message, ok := secretbox.Open(nil, cipher, cursor.Nonce(), cursor.Key())
	if !ok {
		return nil, fmt.Errorf("secretbox decryption failed")
	}
	nextNonce := sha256.Sum256(message)
	cursor.SetNonce(nextNonce[:])
	return message[int(message[0]):], nil
}

type v1State []byte

func (s *v1State) Validate() error {
	expected := v1NonceBytes + v1KeyBytes
	if len(*s) != expected {
		return fmt.Errorf("unexpected state size: %d bytes instead of %d", len(*s), expected)
	}
	return nil
}

func (s *v1State) Nonce() *[v1NonceBytes]byte {
	var nonce [v1NonceBytes]byte
	start := 0
	size := v1NonceBytes
	copy(nonce[:], (*s)[start:start+size])
	return &nonce
}

func (s *v1State) SetNonce(n []byte) {
	copy((*s)[:v1NonceBytes], n)
}

func (s *v1State) Key() *[v1KeyBytes]byte {
	var key [v1KeyBytes]byte
	start := v1NonceBytes
	size := v1KeyBytes
	copy(key[:], (*s)[start:start+size])
	return &key
}
