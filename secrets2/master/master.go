package master

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"

	"secrets/shield"
)

const (
	masterCertTag      = "pond/secrets: master key"
	masterPublicBoxTag = "sendto:master@pond/secrets"
	masterCertLifetime = time.Hour * 24 * 30 * 9
)

// Master key for pond/secrets
type Key struct {
	signer ssh.Signer
	boxkey *shield.Shield
}

// Generate new master key certificate
func NewCertificate(signer ssh.Signer) (*ssh.Certificate, error) {
	seed := make([]byte, 16*1024)
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
		Key:             signer.PublicKey(),
		KeyId:           masterCertTag,
		CertType:        ssh.UserCert,
		Serial:          uint64(now.UnixNano()),
		ValidPrincipals: []string{masterCertTag},
		ValidAfter:      uint64(now.Unix()),
		ValidBefore:     uint64(now.Add(masterCertLifetime).Unix()),
		Reserved:        seed,
		Permissions: ssh.Permissions{
			Extensions: map[string]string{
				masterPublicBoxTag: base64.StdEncoding.EncodeToString(pubkey[:]),
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
	err := checkCert(signer, cert)
	if err != nil {
		return nil, err
	}
	certBoxPubKey, err := base64.StdEncoding.DecodeString(cert.Permissions.Extensions[masterPublicBoxTag])
	if err != nil {
		return nil, err
	}
	pubKey, privKey, err := boxKey(signer, cert.Reserved)
	if err != nil {
		return nil, err
	}
	defer shield.Clean(privKey[:])
	if !bytes.Equal(certBoxPubKey, pubKey[:]) {
		return nil, fmt.Errorf("derived box key does not match the one in certificate")
	}
	boxkey, err := shield.New(signer, privKey[:])
	if err != nil {
		return nil, err
	}
	return &Key{
		signer: signer,
		boxkey: boxkey,
	}, nil
}

// Validate master key certificate
func checkCert(signer ssh.Signer, cert *ssh.Certificate) (err error) {
	err = checkKeyMatch(signer, cert.Key)
	if err != nil {
		return fmt.Errorf("certificate was not given to this key: %w", err)
	}
	err = checkKeyMatch(signer, cert.SignatureKey)
	if err != nil {
		return fmt.Errorf("certificate was not signed by this key: %w", err)
	}
	if len(cert.Reserved) < 1024 {
		return fmt.Errorf("reserved field is too short: %d bytes", len(cert.Reserved))
	}
	if cert.KeyId != masterCertTag {
		return fmt.Errorf("certificate key id is not %q", masterCertTag)
	}
	validator := &ssh.CertChecker{}
	err = validator.CheckCert(masterCertTag, cert)
	if err != nil {
		return err
	}
	return nil
}

// Check if given public and private keys are from the same pair
func checkKeyMatch(private ssh.Signer, public ssh.PublicKey) error {
	var junk [64]byte
	_, err := io.ReadFull(rand.Reader, junk[:])
	if err != nil {
		return err
	}
	sig, err := private.Sign(rand.Reader, junk[:])
	if err != nil {
		return err
	}
	err = public.Verify(junk[:], sig)
	if err != nil {
		return err
	}
	return nil
}