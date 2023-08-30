package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

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
	Expires string   `short:"x" default:"90d" help:"Certificate validity duration"`
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
	err = acl.LoadAdmin(r.AdminCerts())
	if err != nil {
		return "", err
	}
	saddr := os.Getenv("SSH_AUTH_SOCK")
	if saddr == "" {
		return "", fmt.Errorf("environment variable not set: SSH_AUTH_SOCK")
	}
	socket, err := net.Dial("unix", saddr)
	if err != nil {
		return "", err
	}
	defer func() { _ = socket.Close() }()
	agent := agent.NewClient(socket)
	signers, err := agent.Signers()
	if err != nil {
		return "", err
	}
	if len(signers) == 0 {
		return "", fmt.Errorf("no identities available in ssh-agent")
	}
loop_signer:
	for _, signer := range signers {
		for _, capability := range caps {
			for _, p := range c.Path {
				err = acl.Check(signer.PublicKey(), access.Required[capability], p)
				if err != nil {
					continue loop_signer
				}
			}
		}
		cert, err := access.DelegateUser(signer, to, caps, c.Path, c.User, lifetime)
		if err != nil {
			return "", err
		}
		return r.Save(cert)
	}
	return "", fmt.Errorf("ssh-agent: not enough permissions to issue this certificate (tried %d identities)", len(signers))
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
	cert, err := master.Delegate(to, caps, c.Path, c.Admin, lifetime)
	if err != nil {
		return "", err
	}
	return r.Save(cert)
}
