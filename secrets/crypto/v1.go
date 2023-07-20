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
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	v1tag           = 1
	v1sshNonceBytes = 32
	v1kdfNonceBytes = 32
	v1boxNonceBytes = 24
	v1boxKeyBytes   = 32
)

func (s *SecretValue) v1encrypt(sign SignerFunc, value string, keywords ...string) error {
	signature, sshNonce, err := v1signature(sign, keywords)
	if err != nil {
		return err
	}
	key, kdfNonce, err := v1kdf(signature)
	if err != nil {
		return err
	}
	var boxNonce [v1boxNonceBytes]byte
	_, err = io.ReadFull(rand.Reader, boxNonce[:])
	if err != nil {
		return fmt.Errorf("failed to generated nonce for secret box: %w", err)
	}
	var boxKey [v1boxKeyBytes]byte
	n := copy(boxKey[:], key)
	if n != len(key) {
		return fmt.Errorf("failed to copy secret box key into array")
	}
	secret := bytes.Join([][]byte{
		[]byte{v1tag},
		sshNonce,
		kdfNonce,
		boxNonce[:],
	}, nil)
	_ = secretbox.Seal(secret, []byte(value), &boxNonce, &boxKey)
	*s = SecretValue(secret)
	return nil
}

func (s *SecretValue) v1decrypt(sign SignerFunc, keywords ...string) (string, error) {
	return "", nil
}

// Produce a deterministic cryptographic signature for non-secret input
func v1signature(sign SignerFunc, keywords []string) (signature, nonce []byte, err error) {
	nonce = make([]byte, v1sshNonceBytes)
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce for ssh message: %w", err)
	}
	var chunks = make([][]byte, len(keywords)+2)
	chunks[0] = magicHeader
	chunks[1] = nonce
	for index, word := range keywords {
		chunks[index+2] = []byte(word)
	}
	sig, err := sign(bytes.Join(chunks, magicSeparator))
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
func v1kdf(signature []byte) (key, nonce []byte, err error) {
	if len(signature) == 0 {
		return nil, nil, fmt.Errorf("v1kdf: empty input")
	}
	nonce = make([]byte, v1kdfNonceBytes)
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce for key derivation: %w", err)
	}
	kdf := hkdf.New(sha256.New, signature, nonce, magicHeader)

	// Throw alignment off a little
	_, err = io.CopyN(io.Discard, kdf, int64(nonce[0])+int64(signature[0]))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to skip first bytes of HKDF: %w", err)
	}

	key = make([]byte, v1boxKeyBytes)
	_, err = kdf.Read(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key from HKDF: %w", err)
	}
	return key, nonce, nil
}
