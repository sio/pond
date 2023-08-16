package access

import (
	"crypto/rand"
	"errors"
	"time"

	"golang.org/x/crypto/ssh"
)

// Delegate administrative capabilities
func DelegateAdmin(
	from ssh.Signer,
	to ssh.PublicKey,
	caps []Capability,
	paths []string,
	name string,
	lifetime time.Duration,
) (cert *ssh.Certificate, err error) {
	for _, c := range caps {
		if !c.Admin() {
			return nil, errors.New("not allowed to mix administrative and user capabilities")
		}
	}
	return delegate(from, to, caps, paths, name, lifetime)
}

// Delegate user capabilities
func DelegateUser(
	from ssh.Signer,
	to ssh.PublicKey,
	caps []Capability,
	paths []string,
	name string,
	lifetime time.Duration,
) (cert *ssh.Certificate, err error) {
	for _, c := range caps {
		if !c.User() {
			return nil, errors.New("not allowed to mix administrative and user capabilities")
		}
	}
	return delegate(from, to, caps, paths, name, lifetime)
}

func delegate(
	from ssh.Signer,
	to ssh.PublicKey,
	caps []Capability,
	paths []string,
	name string,
	lifetime time.Duration,
) (cert *ssh.Certificate, err error) {
	if len(caps) == 0 {
		return nil, errors.New("no capabilities to delegate")
	}
	if len(paths) == 0 {
		return nil, errors.New("no paths to delegate capabilities over")
	}
	for _, p := range paths {
		if p[0] != '/' {
			return nil, errors.New("relative paths are not allowed in certificates")
		}
	}
	if len(name) == 0 {
		return nil, errors.New("certificate owner name not provided")
	}
	if pubEqual(from.PublicKey(), to) {
		return nil, errors.New("self delegation not allowed")
	}
	capabilities := make(map[string]string)
	for _, c := range caps {
		capabilities[string(c)] = ""
	}
	now := time.Now()
	cert = &ssh.Certificate{
		Key:             to,
		KeyId:           name,
		CertType:        ssh.UserCert,
		Serial:          uint64(now.UnixNano()),
		ValidPrincipals: paths,
		ValidAfter:      uint64(now.Unix()),
		ValidBefore:     uint64(now.Add(lifetime).Unix()),
		Permissions: ssh.Permissions{
			CriticalOptions: capabilities,
		},
	}
	err = cert.SignCert(rand.Reader, from)
	if err != nil {
		return nil, err
	}
	return cert, nil
}
