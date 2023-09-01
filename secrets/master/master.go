package master

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/agent"
)

const (
	masterCertLifetime = time.Hour * 24 * 30 * 9
	masterTag          = "pond/secrets: master key"
)

// Master key for pond/secrets
type Key struct {
	signer ssh.Signer
	seed   []byte
}

// Open master key referenced by a certificate stored on file system
func Open(path string) (*Key, error) {
	cert, err := LoadCertificate(path)
	if err != nil {
		return nil, err
	}
	signer, err := agent.New(cert.PublicKey())
	if err != nil {
		return nil, err
	}
	return NewKey(signer, cert)
}

// Generate new master key certificate
func NewCertificate(signer ssh.Signer) (*Certificate, error) {
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
				masterTag: "",
			},
		},
	}
	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		return nil, err
	}
	return &Certificate{ssh: cert}, nil
}

// Initialize master key from ssh signer and a corresponding certificate
func NewKey(signer ssh.Signer, cert *Certificate) (*Key, error) {
	err := cert.Validate(signer.PublicKey())
	if err != nil {
		return nil, err
	}
	expected, _, err := boxKey(signer, cert.ssh.Reserved)
	if err != nil {
		return nil, err
	}
	actual := cert.SendTo()
	if *expected != *actual {
		return nil, fmt.Errorf("derived box key does not match the one in certificate")
	}
	seed := make([]byte, len(cert.ssh.Reserved))
	copy(seed, cert.ssh.Reserved)
	return &Key{
		signer: signer,
		seed:   seed,
	}, nil
}

var _ ssh.Signer = &Key{}

func (k *Key) PublicKey() ssh.PublicKey {
	return k.signer.PublicKey()
}

func (k *Key) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	return k.signer.Sign(rand, data)
}
