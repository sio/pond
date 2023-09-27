package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	hostCertLifetime       = time.Hour * 24
	hostCertRenewThreshold = time.Hour
)

func ephemeralHostCert(signer ssh.Signer) (ssh.Signer, error) {
	_, ephemeralCryptoKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	ephemeralKey, err := ssh.NewSignerFromKey(ephemeralCryptoKey)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	cert := &ssh.Certificate{
		Key:         ephemeralKey.PublicKey(),
		KeyId:       "secretd host key",
		CertType:    ssh.HostCert,
		Serial:      uint64(now.Unix()),
		ValidAfter:  uint64(now.Unix()) - 1,
		ValidBefore: uint64(now.Add(hostCertLifetime).Unix()),
	}
	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		return nil, err
	}
	certSigner, err := ssh.NewCertSigner(cert, ephemeralKey)
	if err != nil {
		return nil, err
	}
	return certSigner, nil
}
