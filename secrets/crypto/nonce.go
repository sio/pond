package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Cryptographically secure nonce
func randomNonce(size int) (nonce []byte, err error) {
	return makeNonce(size, rand.Reader)
}

// Deterministic, not cryptographically secure nonce
//
// Even though resulting nonce looks random, it is not.
// The amount of entropy is the same as in the provided seed.
func deterministicNonce(size int, seed []byte) (nonce []byte, err error) {
	if size > 255*sha512.Size {
		return nil, errors.New("requested nonce larger than max HKDF output")
	}
	const info = "pond/secrets: deterministic nonce"
	var salt = sha256.Sum256(seed)
	return makeNonce(
		size,
		hkdf.New(sha512.New, seed, salt[:], []byte(info)),
	)
}

func makeNonce(size int, source io.Reader) (nonce []byte, err error) {
	nonce = make([]byte, size)
	_, err = io.ReadFull(source, nonce)
	if err != nil {
		return nil, err
	}
	return nonce, nil
}
