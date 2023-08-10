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
	"golang.org/x/crypto/ssh"

	"secrets/util"
)

const (
	v1tag             byte = 1
	v1sshNonceBytes   int  = 32
	v1kdfNonceBytes   int  = 32
	v1boxNonceBytes   int  = 24
	v1paddingMaxBytes int  = 32
	v1boxKeyBytes     int  = 32
)

func v1encrypt(signer ssh.Signer, path []string, value []byte) (cipher []byte, err error) {
	var nonce = make([]byte, v1sshNonceBytes+v1kdfNonceBytes+v1boxNonceBytes+v1paddingMaxBytes)
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, fmt.Errorf("nonce generation: %w", err)
	}

	var sshNonce, kdfNonce, boxNonce, padding []byte
	var cursor int

	sshNonce = nonce[cursor : cursor+v1sshNonceBytes]
	cursor += v1sshNonceBytes

	kdfNonce = nonce[cursor : cursor+v1kdfNonceBytes]
	cursor += v1kdfNonceBytes

	boxNonce = nonce[cursor : cursor+v1boxNonceBytes]
	cursor += v1boxNonceBytes

	padding = nonce[cursor : cursor+v1paddingMaxBytes]
	padding = padding[:1+int(padding[0])%v1paddingMaxBytes]

	signature, err := v1signature(signer, path, sshNonce)
	if err != nil {
		return nil, fmt.Errorf("ssh signature: %w", err)
	}
	key, err := v1kdf(signature, kdfNonce)
	if err != nil {
		return nil, fmt.Errorf("key derivation: %w", err)
	}
	var boxNonceArray [v1boxNonceBytes]byte
	n := copy(boxNonceArray[:], boxNonce)
	if n != v1boxNonceBytes {
		return nil, fmt.Errorf("copying nonce to array failed")
	}

	secret := bytes.Join([][]byte{
		[]byte{v1tag},
		sshNonce,
		kdfNonce,
		boxNonce,
	}, nil)
	return secretbox.Seal(secret, append(padding, value...), &boxNonceArray, &key), nil
}

func v1decrypt(signer ssh.Signer, path []string, cipher []byte) (value []byte, err error) {
	const minLen = 1 + v1sshNonceBytes + v1kdfNonceBytes + v1boxNonceBytes + secretbox.Overhead
	if len(cipher)-minLen < 1 {
		return nil, fmt.Errorf("encrypted value is too short: got %d bytes (want %d+ bytes)", len(cipher), minLen)
	}
	if cipher[0] != v1tag {
		return nil, fmt.Errorf("version tag mismatch: got %d, want %d", cipher[0], v1tag)
	}

	var cursor int = 1
	var sshNonce, kdfNonce, boxNonce, box []byte

	sshNonce = cipher[cursor : cursor+v1sshNonceBytes]
	cursor += v1sshNonceBytes

	kdfNonce = cipher[cursor : cursor+v1kdfNonceBytes]
	cursor += v1kdfNonceBytes

	boxNonce = cipher[cursor : cursor+v1boxNonceBytes]
	cursor += v1boxNonceBytes

	var boxNonceArray [v1boxNonceBytes]byte
	n := copy(boxNonceArray[:], boxNonce)
	if n != v1boxNonceBytes {
		return nil, fmt.Errorf("copying nonce to array failed")
	}

	box = cipher[cursor:]

	signature, err := v1signature(signer, path, sshNonce)
	if err != nil {
		return nil, err
	}
	key, err := v1kdf(signature, kdfNonce)
	if err != nil {
		return nil, err
	}

	var ok bool
	value, ok = secretbox.Open(nil, box, &boxNonceArray, &key)
	if !ok {
		return nil, fmt.Errorf("secretbox decryption failed")
	}
	if len(value) < 1 {
		return nil, fmt.Errorf("missing padding length tag after decryption")
	}
	skip := 1 + int(value[0])%v1paddingMaxBytes
	if len(value) < skip {
		return nil, fmt.Errorf("invalid padding length (%d bytes) in decrypted value (%d bytes)", skip, len(value))
	}
	return value[skip:], nil
}

// Produce a deterministic cryptographic signature for non-secret input
func v1signature(signer ssh.Signer, path []string, nonce []byte) (signature []byte, err error) {
	var chunks = make([][]byte, len(path)+2)
	chunks[0] = magicHeader
	chunks[1] = nonce
	for index, element := range path {
		chunks[index+2] = []byte(element)
	}
	sig, err := signer.Sign(util.FailingReader, bytes.Join(chunks, magicSeparator))
	if err != nil {
		return nil, fmt.Errorf("ssh signature failed: %w", err)
	}
	signature = sig.Blob
	if len(signature) < 64 {
		return nil, fmt.Errorf("ssh signature too short: %d bytes (want at least 64)", len(signature))
	}
	return signature, nil
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
func v1kdf(signature, nonce []byte) (key [v1boxKeyBytes]byte, err error) {
	var zero [v1boxKeyBytes]byte
	if len(signature) == 0 {
		return zero, fmt.Errorf("v1kdf: empty input")
	}
	kdf := hkdf.New(sha256.New, signature, nonce, magicHeader)

	// Throw alignment off a little
	_, err = io.CopyN(io.Discard, kdf, int64(nonce[0])+int64(signature[0]))
	if err != nil {
		return zero, fmt.Errorf("failed to skip first bytes of HKDF: %w", err)
	}

	_, err = io.ReadFull(kdf, key[:])
	if err != nil {
		return zero, fmt.Errorf("failed to read key from HKDF: %w", err)
	}
	return key, nil
}
