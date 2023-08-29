package master

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/access"
)

const (
	masterCertLifetime = time.Hour * 24 * 30 * 9
)

// Master key for pond/secrets
type Key struct {
	signer ssh.Signer
	seed   []byte
}

// Generate new master key certificate
func NewCertificate(signer ssh.Signer) (*ssh.Certificate, error) {
	// We use random seed larger than any hash used by ssh signatures,
	// prefixed by a constant string to avoid collisions with other uses of ssh
	// signature.
	//
	// Nevertheless, one should not use master key for unrelated signing
	// operations to avoid leaking the signature from which box keys are derived.
	const seedSize = sha512.Size * 4

	seed := make([]byte, seedSize)
	_, err := io.ReadFull(rand.Reader, seed)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	pubkey, _, err := boxKey(signer, seed)
	if err != nil {
		return nil, err
	}
	cert := &ssh.Certificate{
		Key:         signer.PublicKey(),
		KeyId:       base64.StdEncoding.EncodeToString(pubkey[:]),
		CertType:    ssh.UserCert,
		Serial:      uint64(now.UnixNano()),
		ValidAfter:  uint64(now.Unix()),
		ValidBefore: uint64(now.Add(masterCertLifetime).Unix()),
		Reserved:    seed,
		Permissions: ssh.Permissions{
			CriticalOptions: map[string]string{
				string(access.Master): "",
			},
		},
	}
	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// Initialize master key from ssh signer and a corresponding certificate
func NewKey(signer ssh.Signer, cert *ssh.Certificate) (*Key, error) {
	err := access.ValidateMasterCert(signer.PublicKey(), cert)
	if err != nil {
		return nil, err
	}
	certBoxPubKey, err := base64.StdEncoding.DecodeString(cert.KeyId)
	if err != nil {
		return nil, err
	}
	pubkey, _, err := boxKey(signer, cert.Reserved)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(certBoxPubKey, pubkey[:]) {
		return nil, fmt.Errorf("derived box key does not match the one in certificate")
	}
	seed := make([]byte, len(cert.Reserved))
	copy(seed, cert.Reserved)
	return &Key{
		signer: signer,
		seed:   seed,
	}, nil
}
