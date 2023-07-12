package metal_id

import (
	"crypto"
	"encoding/pem"

	"github.com/caarlos0/sshmarshal" // PR pending: https://github.com/golang/go/issues/37132
	"golang.org/x/crypto/ssh"
)

func EncodePrivateKey(key crypto.PrivateKey) ([]byte, error) {
	block, err := sshmarshal.MarshalPrivateKey(key, "")
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(block), nil
}

func EncodePublicKey(key crypto.PublicKey) ([]byte, error) {
	sshkey, err := ssh.NewPublicKey(key)
	if err != nil {
		return nil, err
	}
	return ssh.MarshalAuthorizedKey(sshkey), nil
}
