package main

import (
	"fmt"
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
	fmt.Println(*c)
	return nil
}
