package access

import (
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
) (cert *Certificate, err error) {
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
) (cert *Certificate, err error) {
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
) (cert *Certificate, err error) {
	cert, err = NewCertificate(to, name, paths, caps)
	if err != nil {
		return nil, err
	}
	err = cert.Sign(from, lifetime)
	if err != nil {
		return nil, err
	}
	return cert, nil
}
