package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

var cli struct {
	Chdir string  `short:"C" env:"SECRETS_DIR" placeholder:"path" type:"path" help:"Change working directory prior to executing"`
	Init  InitCmd `cmd:"init" help:"Initialize secrets repository in an empty directory"`
	Cert  CertCmd `cmd:"cert" help:"Issue certificate to delegate user/administrator privileges"`
	Set   SetCmd  `cmd:"set" help:"Set secret value from argument/file/stdin/$EDITOR"`
}

func main() {
	ctx := kong.Parse(&cli)
	if cli.Chdir != "" {
		err := os.Chdir(cli.Chdir)
		if err != nil {
			fail(err)
		}
	}
	err := ctx.Run()
	if err != nil {
		// Remove the name of failed function that kong prepends:
		// https://github.com/alecthomas/kong/blob/074ccd090604a69363b9e6f56b0205bafb79884d/callbacks.go#L134
		_, reason, found := strings.Cut(err.Error(), " ")
		if found {
			fail(reason)
		} else {
			fail(err)
		}
	}
}

func ok(message any, args ...any) {
	out(os.Stdout, message, args...)
}

func warn(message any, args ...any) {
	var buf = new(bytes.Buffer)
	out(buf, message, args...)
	str := buf.String()
	out(os.Stderr, "Warning: %s", str[:len(str)-1])
}

func fail(message any, args ...any) {
	var buf = new(bytes.Buffer)
	out(buf, message, args...)
	errmsg := buf.String()
	if !strings.HasPrefix(strings.ToLower(errmsg), "error") {
		errmsg = "Error: " + errmsg
	}
	out(os.Stderr, errmsg)
	os.Exit(1)
}

func out(dest io.Writer, message any, args ...any) {
	var s string
	var ok bool
	if s, ok = message.(string); !ok {
		_, _ = fmt.Fprintln(dest, message)
		return
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != '\n' && last != '\r' && last != '\t' && last != ' ' {
			break
		}
		s = s[:len(s)-1]
	}
	if len(args) == 0 {
		_, _ = fmt.Fprintln(dest, s)
		return
	}
	_, _ = fmt.Fprintf(dest, s+"\n", args...)
}
