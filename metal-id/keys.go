package metal_id

import (
	"crypto"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

func EncodePrivateKey(key crypto.PrivateKey) ([]byte, error) {
	block, err := ssh.MarshalPrivateKey(key, "")
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
