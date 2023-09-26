package certs

import (
	"testing"

	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"
)

// Check if we can join two trust chains together
//
//	                     SERVER           CLIENT
//	                    (CHAIN 1)        (CHAIN 2)
//	┌─Ephemeral key──────────────────────────────────────────┐
//	│                     Root                               │
//	│                    Cert R1                             │
//	└───────────────────────┬────────────────────────────────┘
//	                        │
//	┌─Client key────────────▼────────────────────────────────┐
//	│                  Intermediate        Root              │
//	│                    Cert I1          Cert R2            │
//	└────────────────────────────────────────┬───────────────┘
//	                                         │
//	┌─Ephemeral key──────────────────────────▼───────────────┐
//	│                                      Leaf              │
//	│                                     Cert L2            │
//	└────────────────────────────────────────────────────────┘
//
// - On server (tlsTrustedPool):
//   - Root CA certificate (R1) is issued for self-signed ephemeral key
//   - Intermediate CA certificates (I1) are issued for each client expected to connect.
//     We can not self sign these because we do not have private keys for clients,
//     only public ones.
//
// - On client (tlsLeafCert):
//   - Root CA certificate (R2) is self-signed by our client private key
//   - Leaf certificate (L2) is issued to an ephemeral key signed by root CA cert
//
// Server will validate L2 received from client by attempting to build chain R1-I1-L2
func TestPKI(t *testing.T) {
	clientKey, err := PrivateKey("testkeys/alice")
	if err != nil {
		t.Fatal(err)
	}
	clientCert, err := tlsLeafCert(time.Hour, clientKey)
	if err != nil {
		t.Fatal(err)
	}

	client, err := PublicKey("testkeys/alice.pub")
	if err != nil {
		t.Fatal(err)
	}
	trusted, err := tlsTrustedPool(time.Hour, client)
	if err != nil {
		t.Fatal(err)
	}
	verifyOpts := x509.VerifyOptions{
		Roots:         x509.NewCertPool(),
		Intermediates: x509.NewCertPool(),
	}
	verifyOpts.Roots.AddCert(trusted[0])
	for _, cert := range trusted[1:] {
		verifyOpts.Intermediates.AddCert(cert)
	}
	for index, cert := range trusted {
		_, err = cert.Verify(verifyOpts)
		if err != nil {
			t.Fatalf("verifying initial trust store, certificate %d: %v", index, err)
		}
	}
	_, err = clientCert.Verify(verifyOpts)
	if err != nil {
		t.Fatalf("verifying client cert: %v", err)
	}
}

// Issue a leaf certificate for the given private key
func tlsLeafCert(lifetime time.Duration, key crypto.Signer) (*x509.Certificate, error) {
	keyRepr, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		return nil, err
	}
	rootCert := &x509.Certificate{
		IsCA:         true,
		SerialNumber: big.NewInt(10),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(lifetime),
		Subject: pkix.Name{
			Organization:       []string{"pond/nbd"},
			OrganizationalUnit: []string{base64.StdEncoding.EncodeToString(keyRepr)},
		},
	}
	rootDer, err := x509.CreateCertificate(rand.Reader, rootCert, rootCert, key.Public(), key)
	if err != nil {
		return nil, err
	}
	rootCert, err = x509.ParseCertificate(rootDer)
	if err != nil {
		return nil, err
	}
	leafPubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	leafCert := &x509.Certificate{
		IsCA:         false,
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(lifetime),
		Issuer:       rootCert.Subject,
	}
	leafDer, err := x509.CreateCertificate(rand.Reader, leafCert, rootCert, leafPubKey, key)
	if err != nil {
		return nil, err
	}
	leafCert, err = x509.ParseCertificate(leafDer)
	if err != nil {
		return nil, err
	}
	return leafCert, nil
}

// Build a pool of trusted intermediate TLS certificates
// based on the list of preapproved public keys
func tlsTrustedPool(lifetime time.Duration, trusted ...crypto.PublicKey) (certs []*x509.Certificate, err error) {
	_, ephemeralRootKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	ephemeralRootCert := &x509.Certificate{
		IsCA:         true,
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(lifetime),
		Subject: pkix.Name{
			Organization: []string{"pond/nbd"},
		},
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	ephemeralDer, err := x509.CreateCertificate(
		rand.Reader,
		ephemeralRootCert,
		ephemeralRootCert,
		ephemeralRootKey.Public(),
		ephemeralRootKey,
	)
	if err != nil {
		return nil, err
	}
	ephemeralRootCert, err = x509.ParseCertificate(ephemeralDer)
	if err != nil {
		return nil, err
	}
	intermediateCertTemplate := &x509.Certificate{
		IsCA:      true,
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(lifetime),
		Subject: pkix.Name{
			Organization:       []string{"pond/nbd"},
			OrganizationalUnit: make([]string, 1),
		},
		Issuer: pkix.Name{
			Organization: []string{"pond/nbd"},
		},
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	certs = make([]*x509.Certificate, 0, 1+len(trusted))
	certs = append(certs, ephemeralRootCert)
	for index, publicKey := range trusted {
		intermediateCertTemplate.SerialNumber = big.NewInt(int64(1 + index))
		keyRepr, err := x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return nil, fmt.Errorf("certificate #%d: %w", index, err)
		}
		intermediateCertTemplate.Subject.OrganizationalUnit[0] = base64.StdEncoding.EncodeToString(keyRepr)
		intermediateDer, err := x509.CreateCertificate(
			rand.Reader,
			intermediateCertTemplate,
			ephemeralRootCert,
			publicKey,
			ephemeralRootKey,
		)
		if err != nil {
			return nil, fmt.Errorf("certificate #%d: %w", index, err)
		}
		cert, err := x509.ParseCertificate(intermediateDer)
		if err != nil {
			return nil, fmt.Errorf("certificate #%d: %w", index, err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}
