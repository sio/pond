package util

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
)

// Load ssh public key from file system
func LoadPublicKey(path string) (ssh.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	if err != nil {
		return nil, err
	}
	if cert, ok := key.(*ssh.Certificate); ok {
		return cert.Key, nil
	}
	return key, nil
}

// Load ssh certificate from file system
func LoadCertificate(path string) (*ssh.Certificate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(raw)
	if err != nil {
		return nil, err
	}
	cert, ok := key.(*ssh.Certificate)
	if !ok {
		return nil, fmt.Errorf("not an ssh-certificate (%s): %s", key.Type(), path)
	}
	return cert, nil
}
