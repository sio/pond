package master

import (
	"crypto/sha512"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ssh"

	"secrets/shield"
	"secrets/util"
)

// Derive NaCl box key pair from a deterministic SSH signer
// and random non-secret nonce
func boxKey(signer ssh.Signer, nonce []byte) (public, private *[32]byte, err error) {
	if len(nonce) < sha512.Size {
		return nil, nil, errors.New("nonce is too short")
	}
	signature, err := signer.Sign(util.AntiReader, append([]byte(masterCertTag), nonce...))
	if err == util.ErrAntiReader {
		return nil, nil, fmt.Errorf("signature not deterministic: %s", signer.PublicKey().Type())
	}
	if err != nil {
		return nil, nil, err
	}
	salt := sha512.Sum512(signature.Blob)
	seed := argon2.IDKey(signature.Blob, salt[:], 4, 256*1024, 2, 32)

	private = new([32]byte)
	copy(private[:], seed)
	shield.Clean(seed)

	public = deriveBoxKey(private)
	return public, private, nil
}

// Derive NaCl box public key from a private seed (random bytes)
func deriveBoxKey(seed *[32]byte) (public *[32]byte) {
	public = new([32]byte)
	curve25519.ScalarBaseMult(public, seed)
	return public
}
