package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"

	"secrets/agent"
	"secrets/master"
)

type InitCmd struct {
	PublicKey string `arg:"" name:"pubkey" type:"path" help:"Path to SSH public key to be used as repository master key"`
}

func (c *InitCmd) Run() error {
	err := checkRepoEmpty()
	if err != nil {
		return err
	}
	signer, err := agent.Open(c.PublicKey)
	if err != nil {
		return err
	}
	cert, err := master.NewCertificate(signer)
	if err != nil {
		return err
	}
	certPath, err := filepath.Abs(filepath.Join("access", "master.cert"))
	if err != nil {
		return err
	}
	for _, subdir := range []string{"access", "secrets"} {
		err = os.Mkdir(subdir, 0700)
		if err != nil {
			return err
		}
	}
	ok("Created directories for secrets repository")
	file, err := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	_, err = file.Write(ssh.MarshalAuthorizedKey(cert))
	if err != nil {
		return err
	}
	ok("Saved master certificate: %s", certPath)
	ok("Secrets repository initialized successfully")
	return nil
}

func checkRepoEmpty() error {
	items, err := os.ReadDir(".")
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Name() == ".git" {
			continue
		}
		if strings.HasSuffix(item.Name(), ".md") {
			continue
		}
		dir, err := filepath.Abs(".")
		if err != nil {
			dir = "."
		}
		return fmt.Errorf("directory not empty: %s", dir)
	}
	return nil
}
