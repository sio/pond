package master

import (
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/access"
	"github.com/sio/pond/secrets/util"
)

// Derive NaCl box key pair from a deterministic SSH signer
// and random non-secret nonce
//
// HKDF is fast enough for us to never store the resulting key for longer than
// it's used.
//
// Bruteforcing is not a concern: SSH signature is a least 64B long (ed25519,
// rsa even longer), so it's more rational for an attacker to try to bruteforce
// the 32B box private key instead of bruteforcing a signature from which a valid
// key may be derived.
func boxKey(signer ssh.Signer, nonce []byte) (public, private *[32]byte, err error) {
	if len(nonce) < sha512.Size {
		return nil, nil, errors.New("nonce is too short")
	}
	signature, err := signer.Sign(util.AntiReader, append([]byte(access.Master), nonce...))
	if err == util.ErrAntiReader {
		return nil, nil, fmt.Errorf("signature not deterministic: %s", signer.PublicKey().Type())
	}
	if err != nil {
		return nil, nil, err
	}
	salt := sha512.Sum512(signature.Blob)
	kdf := hkdf.New(sha512.New, signature.Blob, salt[:], []byte(access.Master))
	private = new([32]byte)
	_, err = io.ReadFull(kdf, private[:])
	clean(signature.Blob)
	if err != nil {
		return nil, nil, fmt.Errorf("HKDF: %w", err)
	}
	public = derivePublicKey(private)
	return public, private, nil
}

// Derive NaCl box public key from a private seed (random bytes)
func derivePublicKey(seed *[32]byte) (public *[32]byte) {
	public = new([32]byte)
	curve25519.ScalarBaseMult(public, seed)
	return public
}

func clean(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
	_, _ = rand.Read(b)
}
