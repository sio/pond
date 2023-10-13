package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	hostCertLifetime       = time.Hour * 24
	hostCertRenewThreshold = time.Hour
)

func ephemeralHostCert(signer ssh.Signer) (s ssh.Signer, expires time.Time, err error) {
	var zero time.Time
	_, ephemeralCryptoKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, zero, err
	}
	ephemeralKey, err := ssh.NewSignerFromKey(ephemeralCryptoKey)
	if err != nil {
		return nil, zero, err
	}
	now := time.Now()
	expires = now.Add(hostCertLifetime)
	cert := &ssh.Certificate{
		Key:         ephemeralKey.PublicKey(),
		KeyId:       "secretd host key",
		CertType:    ssh.HostCert,
		Serial:      uint64(now.Unix()),
		ValidAfter:  uint64(now.Unix()) - 1,
		ValidBefore: uint64(expires.Unix()),
	}
	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		return nil, zero, err
	}
	certSigner, err := ssh.NewCertSigner(cert, ephemeralKey)
	if err != nil {
		return nil, zero, err
	}
	return certSigner, expires, nil
}

// Background function that renews host ssh certificate on schedule
func (s *Server) renewHostCert(ctx context.Context) {
	var cert ssh.Signer
	var expires time.Time
	var err error
	for {
		for {
			cert, expires, err = ephemeralHostCert(s.master)
			if err == nil {
				break
			}
			s.log("Failed to renew host ssh certificate: %v", err)
			select {
			case <-time.After(time.Second * 5):
				// retry
			case <-ctx.Done():
				return
			}
		}
		s.sshMu.Lock()
		s.ssh.AddHostKey(cert)
		s.sshMu.Unlock()
		select {
		case <-time.After(time.Until(expires.Add(-hostCertRenewThreshold))):
			// renew again
		case <-ctx.Done():
			return
		}
	}
}
