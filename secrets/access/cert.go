package access

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/util"
)

// Use ssh certificate to store access control information
type Certificate struct {
	ssh *ssh.Certificate
}

// Create new user certificate
//
// Returned certificate will be unsigned, to make it usable call Sign() method
func NewCertificate(recepient ssh.PublicKey, name string, paths []string, caps []Capability) (*Certificate, error) {
	if len(caps) == 0 {
		return nil, errors.New("no capabilities provided")
	}
	if len(paths) == 0 {
		return nil, errors.New("no paths provided")
	}
	for _, p := range paths {
		if p[0] != '/' {
			return nil, errors.New("relative paths are not allowed")
		}
	}
	if len(name) == 0 {
		return nil, errors.New("certificate owner name not provided")
	}
	capabilities := make(map[string]string)
	for _, c := range caps {
		capabilities[string(c)] = ""
	}
	ssh := &ssh.Certificate{
		Key:             recepient,
		KeyId:           name,
		CertType:        ssh.UserCert,
		ValidPrincipals: paths,
		Permissions: ssh.Permissions{
			CriticalOptions: capabilities,
		},
	}
	return &Certificate{ssh}, nil
}

// Load user certificate from file system
func LoadCertificate(path string) (*Certificate, error) {
	ssh, err := util.LoadCertificate(path)
	if err != nil {
		return nil, err
	}
	cert := &Certificate{ssh}
	err = cert.Validate()
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// Recepient public key
func (c *Certificate) PublicKey() ssh.PublicKey {
	return c.ssh.Key
}

// Issuer public key
func (c *Certificate) SignatureKey() ssh.PublicKey {
	return c.ssh.SignatureKey
}

// Recepient user name
func (c *Certificate) Name() string {
	return c.ssh.KeyId
}

// List of paths to which access is granted
func (c *Certificate) Paths() []string {
	return c.ssh.ValidPrincipals
}

// Validity period start
func (c *Certificate) ValidAfter() uint64 {
	return c.ssh.ValidAfter
}

// Validity period end
func (c *Certificate) ValidBefore() uint64 {
	return c.ssh.ValidBefore
}

// Which capabilities are granted by this certificate
func (c *Certificate) Capabilities() []Capability {
	var caps = make([]Capability, len(c.ssh.Permissions.CriticalOptions))
	var index int
	for item := range c.ssh.Permissions.CriticalOptions {
		caps[index] = Capability(item)
		index++
	}
	if len(caps) == 0 {
		caps = nil
	}
	return caps
}

// Check whether this is an administrator certificate. Always call Validate() first!
func (c *Certificate) Admin() bool {
	if c.ssh == nil || c.ssh.SignatureKey == nil {
		return false
	}
	caps := c.Capabilities()
	if len(caps) == 0 {
		return false
	}
	return caps[0].Admin()
}

// Sign user certificate
func (c *Certificate) Sign(authority ssh.Signer, lifetime time.Duration) error {
	if pubEqual(c.ssh.Key, authority.PublicKey()) {
		return errors.New("self delegation not allowed")
	}
	now := time.Now()
	c.ssh.Serial = uint64(now.Unix())
	c.ssh.ValidAfter = uint64(now.Unix())
	c.ssh.ValidBefore = uint64(now.Add(lifetime).Unix())
	err := c.ssh.SignCert(rand.Reader, authority)
	if err != nil {
		return err
	}
	return c.Validate()
}

// Convert certificate to ssh file format
func (c *Certificate) Marshal() []byte {
	if c.ssh == nil {
		return nil
	}
	return ssh.MarshalAuthorizedKey(c.ssh)
}

// Validate user certificate
func (c *Certificate) Validate() error {
	if len(c.ssh.KeyId) == 0 {
		return fmt.Errorf("empty key id")
	}
	if len(c.ssh.ValidPrincipals) == 0 {
		return fmt.Errorf("no allowed paths listed in principals field")
	}
	for _, p := range c.ssh.ValidPrincipals {
		if p[0] != '/' {
			return fmt.Errorf("relative paths not allowed in principals field: %q", p)
		}
	}
	if len(c.ssh.Permissions.CriticalOptions) == 0 {
		return fmt.Errorf("no capabilities listed in critical options field")
	}
	var admin *bool
	for _, capability := range c.Capabilities() {
		if !capability.Valid() {
			return fmt.Errorf("invalid capability: %s", capability)
		}
		if admin == nil {
			admin = new(bool)
			*admin = capability.Admin()
		}
		if *admin != capability.Admin() {
			return fmt.Errorf("mixing user and administrator capabilities in one certificate")
		}
	}
	var supported = make([]string, 2)
	if *admin {
		supported[0] = string(ManageReaders)
		supported[1] = string(ManageWriters)
	} else {
		supported[0] = string(Read)
		supported[1] = string(Write)
	}
	validator := &ssh.CertChecker{
		SupportedCriticalOptions: supported,
	}
	err := validator.CheckCert(c.ssh.ValidPrincipals[0], c.ssh)
	if err != nil {
		return err
	}
	return nil
}
