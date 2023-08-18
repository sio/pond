package access

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// Validate master key certificate
func ValidateMasterCert(master ssh.PublicKey, cert *ssh.Certificate) (err error) {
	if !pubEqual(master, cert.Key) {
		return fmt.Errorf("certificate was not given to this key")
	}
	if !pubEqual(master, cert.SignatureKey) {
		return fmt.Errorf("certificate was not signed by this key")
	}
	if len(cert.KeyId) == 0 {
		return fmt.Errorf("certificate key ID is empty")
	}
	_, err = base64.StdEncoding.DecodeString(cert.KeyId)
	if err != nil {
		return fmt.Errorf("base64: decoding key ID: %w", err)
	}
	if len(cert.Reserved) < sha512.Size {
		return fmt.Errorf("reserved field is too short: %d bytes", len(cert.Reserved))
	}
	if len(cert.ValidPrincipals) != 0 {
		return fmt.Errorf("master key certificate must not list any principals")
	}
	validator := &ssh.CertChecker{
		SupportedCriticalOptions: []string{string(Master)},
	}
	err = validator.CheckCert(string(Master), cert)
	if err != nil {
		return err
	}
	return nil
}

// Check if two public keys are the same
func pubEqual(a, b ssh.PublicKey) bool {
	if a.Type() != b.Type() {
		return false
	}
	return bytes.Equal(a.Marshal(), b.Marshal()) // TODO: key comments are not stripped and may result in false negative
}
