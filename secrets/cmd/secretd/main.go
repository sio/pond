package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/sio/pond/secrets/server"
)

var cli struct {
	Chdir  string `short:"C" env:"SECRETS_DIR" placeholder:"path" type:"path" help:"Change working directory prior to executing"`
	Listen string `short:"l" env:"SECRETS_BIND" default:"tcp://127.0.0.1:20002" placeholder:"address" help:"Address for secretd to bind to, e.g. tcp://10.0.0.123:345 or unix:///var/run/secretd.socket (default: ${default})"`
}

func main() {
	kong.Parse(&cli)
	if cli.Chdir != "" {
		err := os.Chdir(cli.Chdir)
		if err != nil {
			fail(err)
		}
	}
	err := server.Run(cli.Listen, ".")
	if err != nil {
		fail(err)
	}
}

func fail(x ...any) {
	_, _ = fmt.Fprintln(os.Stderr, append([]any{"Error:"}, x...)...)
	os.Exit(1)
}
