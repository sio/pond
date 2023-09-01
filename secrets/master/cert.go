package master

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/util"
)

// Master key certificate
type Certificate struct {
	ssh    *ssh.Certificate
	sendto *[32]byte
}

// Load master key certificate from file system
func LoadCertificate(path string) (*Certificate, error) {
	cert, err := util.LoadCertificate(path)
	if err != nil {
		return nil, err
	}
	master := &Certificate{ssh: cert}
	err = master.Validate(nil)
	if err != nil {
		return nil, err
	}
	return master, nil
}

var _ ssh.PublicKey = &Certificate{}

func (c *Certificate) PublicKey() ssh.PublicKey {
	return c.ssh.Key
}

func (c *Certificate) Type() string {
	return c.ssh.Key.Type()
}

// Convert certificate to ssh file format
func (c *Certificate) Marshal() []byte {
	if c.ssh == nil {
		return nil
	}
	return ssh.MarshalAuthorizedKey(c.ssh)
}

func (c *Certificate) Verify(data []byte, sig *ssh.Signature) error {
	return c.ssh.Key.Verify(data, sig)
}

// NaCl box public key for sending secrets to master
func (c *Certificate) SendTo() *[32]byte {
	if c.sendto != nil {
		var output = new([32]byte)
		*output = *c.sendto
		return output
	}
	decoded, err := base64.StdEncoding.DecodeString(c.ssh.KeyId)
	if err != nil {
		return nil
	}
	c.sendto = new([32]byte)
	copy(c.sendto[:], decoded)
	return c.SendTo()
}

// Validate master key certificate
func (c *Certificate) Validate(pubkey ssh.PublicKey) error {
	if pubkey == nil {
		pubkey = c.ssh.Key
	}
	if !util.EqualSSH(pubkey, c.ssh.Key) {
		return fmt.Errorf("certificate was not given to this key")
	}
	if !util.EqualSSH(pubkey, c.ssh.SignatureKey) {
		return fmt.Errorf("certificate was not signed by this key")
	}
	if len(c.ssh.KeyId) == 0 {
		return fmt.Errorf("certificate key ID is empty")
	}
	_, err := base64.StdEncoding.DecodeString(c.ssh.KeyId)
	if err != nil {
		return fmt.Errorf("base64: decoding key ID: %w", err)
	}
	if len(c.ssh.Reserved) < sha512.Size {
		return fmt.Errorf("reserved field is too short: %d bytes", len(c.ssh.Reserved))
	}
	if len(c.ssh.ValidPrincipals) != 0 {
		return fmt.Errorf("master key certificate must not list any principals")
	}
	if len(c.ssh.Permissions.CriticalOptions) == 0 {
		return fmt.Errorf("critical options field is empty")
	}
	validator := &ssh.CertChecker{
		SupportedCriticalOptions: []string{masterTag},
	}
	err = validator.CheckCert(masterTag, c.ssh)
	if err != nil {
		return err
	}
	return nil
}
