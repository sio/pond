package main

import (
	"github.com/sio/pond/secrets/agent"
	"github.com/sio/pond/secrets/master"
	"github.com/sio/pond/secrets/repo"
)

type InitCmd struct {
	PublicKey string `arg:"" name:"pubkey" type:"path" help:"Path to public part of ssh keypair to be used as repository master key"`
}

func (c *InitCmd) Run() error {
	signer, err := agent.Open(c.PublicKey)
	if err != nil {
		return err
	}
	cert, err := master.NewCertificate(signer)
	if err != nil {
		return err
	}
	repo, err := repo.Create(".", cert)
	if err != nil {
		return err
	}
	ok("Initialized new secrets repository: %s", repo)
	return nil
}
