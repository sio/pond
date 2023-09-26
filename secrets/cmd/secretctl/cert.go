package main

import (
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/secrets/access"
	"github.com/sio/pond/secrets/master"
	"github.com/sio/pond/secrets/repo"
	"github.com/sio/pond/secrets/util"
)

type CertCmd struct {
	User    string   `xor:"class" short:"u" required:"" placeholder:"name" help:"Human readable user identifier"`
	Admin   string   `xor:"class" short:"a" placeholder:"name" help:"Human readable administrator identifier"`
	Key     string   `type:"path" short:"k" required:"" placeholder:"path" help:"Public key of recipient"`
	Read    bool     `short:"r" help:"Read access flag"`
	Write   bool     `short:"w" help:"Write access flag"`
	Expires string   `short:"x" default:"90d" help:"Certificate validity duration (default: ${default})"`
	Path    []string `arg:"" name:"prefix" required:"" help:"List of path prefixes to delegate privileges over"`
}

func (c *CertCmd) Run() error {
	recepient, err := util.LoadPublicKey(c.Key)
	if err != nil {
		return err
	}
	for _, p := range c.Path {
		if len(p) == 0 {
			return fmt.Errorf("empty paths not allowed")
		}
		if p[0] != '/' {
			return fmt.Errorf("relative paths not allowed: %s", p)
		}
	}
	if len(c.Path) == 0 {
		return fmt.Errorf("empty path list not allowed")
	}
	lifetime, err := util.ParseDuration(c.Expires)
	if err != nil {
		return err
	}
	repo, err := repo.Open(".")
	if err != nil {
		return err
	}
	var path string
	switch {
	case c.User != "":
		path, err = c.delegateUser(repo, recepient, lifetime)
	case c.Admin != "":
		path, err = c.delegateAdmin(repo, recepient, lifetime)
	default:
		return fmt.Errorf("either --user or --admin must be provided")
	}
	if err != nil {
		return err
	}
	ok("Issued new certificate: %s", path)
	return nil
}

func (c *CertCmd) delegateUser(r *repo.Repository, to ssh.PublicKey, lifetime time.Duration) (path string, err error) {
	caps := make([]access.Capability, 0, 2)
	if c.Read {
		caps = append(caps, access.Read)
	}
	if c.Write {
		caps = append(caps, access.Write)
	}
	if len(caps) == 0 {
		return "", fmt.Errorf("at least one capability must be provided: --read, --write")
	}
	acl, err := access.Open(r.MasterCert())
	if err != nil {
		return "", err
	}
	warnings, err := acl.LoadAdmin(r.AdminCerts())
	if err != nil {
		return "", err
	}
	for _, w := range warnings {
		warn(w)
	}
	signer, err := acl.FindAgent(c.Path, caps...)
	if err != nil {
		return "", err
	}
	defer func() { _ = signer.Close() }()
	cert, err := access.DelegateUser(signer, to, caps, c.Path, c.User, lifetime)
	if err != nil {
		return "", err
	}
	return r.Save(cert)
}

func (c *CertCmd) delegateAdmin(r *repo.Repository, to ssh.PublicKey, lifetime time.Duration) (path string, err error) {
	caps := make([]access.Capability, 0, 2)
	if c.Read {
		caps = append(caps, access.ManageReaders)
	}
	if c.Write {
		caps = append(caps, access.ManageWriters)
	}
	if len(caps) == 0 {
		return "", fmt.Errorf("at least one capability must be provided: --read, --write")
	}
	master, err := master.Open(r.MasterCert())
	if err != nil {
		return "", err
	}
	cert, err := access.DelegateAdmin(master, to, caps, c.Path, c.Admin, lifetime)
	if err != nil {
		return "", err
	}
	return r.Save(cert)
}
