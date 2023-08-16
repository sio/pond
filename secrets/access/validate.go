package access

import (
	"bytes"
	"crypto/sha512"
	"fmt"

	"golang.org/x/crypto/ssh"
)

const (
	MasterCertTag      = "pond/secrets: master key"
	MasterPublicBoxTag = "sendto:master@pond/secrets"
)

// Validate master key certificate
func ValidateMasterCert(master ssh.PublicKey, cert *ssh.Certificate) (err error) {
	if !pubEqual(master, cert.Key) {
		return fmt.Errorf("certificate was not given to this key")
	}
	if !pubEqual(master, cert.SignatureKey) {
		return fmt.Errorf("certificate was not signed by this key")
	}
	if len(cert.Reserved) < sha512.Size {
		return fmt.Errorf("reserved field is too short: %d bytes", len(cert.Reserved))
	}
	if cert.KeyId != MasterCertTag {
		return fmt.Errorf("certificate key id is not %q", MasterCertTag)
	}
	if len(cert.ValidPrincipals) == 0 {
		return fmt.Errorf("certificate does not contain a list of valid principals")
	}
	validator := &ssh.CertChecker{
		SupportedCriticalOptions: []string{MasterPublicBoxTag},
	}
	err = validator.CheckCert(MasterCertTag, cert)
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
