package certs

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"crypto/ed25519"
	"crypto/rand"
	"math/big"
	"os"
	"time"
)

func TestSelfSigned(t *testing.T) {
	template := &x509.Certificate{
		IsCA:         false,
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24),
		Subject: pkix.Name{
			Country:    []string{"RU"},
			CommonName: "TEST LEAF CERTIFICATE",
		},
	}

	recepient, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519: generating recepient key pair: %v", err)
	}

	_, ca, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519: generating CA key pair: %v", err)
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, recepient, ca)
	if err != nil {
		t.Fatalf("x509: %v", err)
	}

	// Save cert for further inspection
	experimental(t)
	temp, err := os.CreateTemp("", "selfsigned")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = temp.Close() }()
	_, err = temp.Write(der)
	if err != nil {
		t.Fatalf("temp.Write: %v", err)
	}
	t.Logf("Saved self-signed certificate to %s", temp.Name())
}
