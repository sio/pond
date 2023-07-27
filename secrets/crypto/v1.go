//
// Original format of encrypted value storage
//

package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	v1tag             = 1
	v1sshNonceBytes   = 32
	v1kdfNonceBytes   = 32
	v1boxNonceBytes   = 24
	v1paddingMaxBytes = 32
	v1boxKeyBytes     = 32
)

func (s *SecretValue) v1encrypt(signer ssh.Signer, value string, keywords ...string) error {
	signature, sshNonce, err := v1signature(signer, keywords, nil)
	if err != nil {
		return err
	}
	key, kdfNonce, err := v1kdf(signature, nil)
	if err != nil {
		return err
	}
	var boxNonce [v1boxNonceBytes]byte
	_, err = io.ReadFull(rand.Reader, boxNonce[:])
	if err != nil {
		return fmt.Errorf("failed to generate nonce for secret box: %w", err)
	}
	padding := make([]byte, v1paddingMaxBytes)
	_, err = io.ReadFull(rand.Reader, padding)
	if err != nil {
		return fmt.Errorf("failed to generate padding for secret box: %w", err)
	}
	padding = padding[:1+int(padding[0])%v1paddingMaxBytes]
	secret := bytes.Join([][]byte{
		[]byte{v1tag},
		sshNonce,
		kdfNonce,
		boxNonce[:],
	}, nil)
	*s = SecretValue(secretbox.Seal(secret, append(padding, []byte(value)...), &boxNonce, &key))
	return nil
}

func (s *SecretValue) v1decrypt(signer ssh.Signer, keywords ...string) (string, error) {
	const minLen = 1 + v1sshNonceBytes + v1kdfNonceBytes + v1boxNonceBytes + secretbox.Overhead
	if len(*s)-minLen < 1 {
		return "", fmt.Errorf("encrypted value is too short: got %d bytes (want %d+ bytes)", len(*s), minLen)
	}
	if (*s)[0] != v1tag {
		return "", fmt.Errorf("version tag mismatch: got %d, want %d", (*s)[0], v1tag)
	}

	var start, stop int
	var sshNonce, kdfNonce, box, value []byte

	start = 1
	stop = start + v1sshNonceBytes
	sshNonce = []byte((*s)[start:stop])

	start = stop
	stop = start + v1kdfNonceBytes
	kdfNonce = []byte((*s)[start:stop])

	start = stop
	stop = start + v1boxNonceBytes
	var boxNonce [v1boxNonceBytes]byte
	n := copy(boxNonce[:], []byte((*s)[start:stop]))
	if n != len(boxNonce) {
		return "", fmt.Errorf("copying nonce to array failed")
	}

	box = []byte((*s)[stop:])

	signature, _, err := v1signature(signer, keywords, sshNonce)
	if err != nil {
		return "", err
	}
	key, _, err := v1kdf(signature, kdfNonce)
	if err != nil {
		return "", err
	}

	var ok bool
	value, ok = secretbox.Open(nil, box, &boxNonce, &key)
	if !ok {
		return "", fmt.Errorf("secretbox decryption failed")
	}
	if len(value) < 1 {
		return "", fmt.Errorf("missing padding length tag after decryption")
	}
	skip := 1 + int(value[0])%v1paddingMaxBytes
	if len(value) < skip {
		return "", fmt.Errorf("invalid padding length (%d bytes) in decrypted value (%d bytes)", skip, len(value))
	}
	return string(value[skip:]), nil
}

// Produce a deterministic cryptographic signature for non-secret input
func v1signature(signer ssh.Signer, keywords []string, nonce []byte) (signature, nonce_ []byte, err error) {
	if nonce == nil {
		nonce = make([]byte, v1sshNonceBytes)
		_, err = io.ReadFull(rand.Reader, nonce)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate nonce for ssh message: %w", err)
		}
	}
	var chunks = make([][]byte, len(keywords)+2)
	chunks[0] = magicHeader
	chunks[1] = nonce
	for index, word := range keywords {
		chunks[index+2] = []byte(word)
	}
	sig, err := Sign(signer, bytes.Join(chunks, magicSeparator))
	if err != nil {
		return nil, nil, fmt.Errorf("ssh signature failed: %w", err)
	}
	signature = sig.Blob
	if len(signature) < 64 {
		return nil, nil, fmt.Errorf("signature is too short: %d bytes (expected at least 64)", len(signature))
	}
	return signature, nonce, nil
}

// Derive an encryption key from SSH signature
//
// HKDF is a good fit for this:
//   - It is fast and reasonably secure
//   - It was designed for a very similar use case, key derivation from shared
//     secret in Diffie-Hellman exchange:
//     https://www.rfc-editor.org/rfc/rfc5869.html#section-4
//   - It's cheap on resources. Unlike with scrypt and argon2 we do not need to
//     introduce artificial barriers to bruteforcing because input already
//     contains high entropy, even though this entropy may not be distributed
//     uniformly. Bruteforcing a 64 byte binary input is a lot less feasible than
//     bruteforcing a short user generated password.
//     On the other hand we expect multiple users to retrieve multiple secret
//     values simultaneously - spending 64MB of server RAM on each individual
//     value (as suggested by argon2) gets expensive very fast.
func v1kdf(signature, nonce []byte) (key [v1boxKeyBytes]byte, nonce_ []byte, err error) {
	var zero [v1boxKeyBytes]byte
	if len(signature) == 0 {
		return zero, nil, fmt.Errorf("v1kdf: empty input")
	}
	if nonce == nil {
		nonce = make([]byte, v1kdfNonceBytes)
		_, err = io.ReadFull(rand.Reader, nonce)
		if err != nil {
			return zero, nil, fmt.Errorf("failed to generate nonce for key derivation: %w", err)
		}
	}
	kdf := hkdf.New(sha256.New, signature, nonce, magicHeader)

	// Throw alignment off a little
	_, err = io.CopyN(io.Discard, kdf, int64(nonce[0])+int64(signature[0]))
	if err != nil {
		return zero, nil, fmt.Errorf("failed to skip first bytes of HKDF: %w", err)
	}

	_, err = io.ReadFull(kdf, key[:])
	if err != nil {
		return zero, nil, fmt.Errorf("failed to read key from HKDF: %w", err)
	}
	return key, nonce, nil
}
