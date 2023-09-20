package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type SetCmd struct {
	Dest  string `arg:"" name:"secret" help:"Path to secret in repository"`
	Value string `xor:"v" arg:"" optional:"" name:"value" help:"Use CLI argument as plaintext value (optional)"`
	File  string `xor:"v" short:"f" placeholder:"path" type:"existingfile" help:"Use file contents as value (default: read standard input)"`
}

func (c *SetCmd) Run() error {
	if c.Value != "" && c.File != "" {
		return fmt.Errorf("only one of <value> and <--file> is expected to be provided")
	}
	var value string
	switch {
	case c.Value != "":
		// Plain value from CLI args
		value = c.Value
	case c.File != "" && c.File != "-":
		// Value from file
		raw, err := os.ReadFile(c.File)
		if err != nil {
			return err
		}
		value = string(raw)
	default:
		// Read from stdin
		stat, err := os.Stdin.Stat()
		if err != nil {
			return err
		}
		if stat.Mode()&os.ModeNamedPipe != 0 { // https://stackoverflow.com/a/26567513
			raw, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			value = string(raw)
		}
		if value != "" {
			break
		}
		if c.File == "-" {
			return fmt.Errorf("failed to read value from standard input")
		}

		// Fall back to $EDITOR
		value, err = readFromEditor()
		if err != nil {
			return err
		}
	}
	return set(c.Dest, value)
}

func set(path, value string) error {
	fmt.Printf("Path=%s\nValue=%s\n", path, value)
	return nil
}

func readFromEditor() (value string, err error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
		out(os.Stderr, "$EDITOR not set, defaulting to %q", editor)
	}
	tempdir, err := os.MkdirTemp("", "secret")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tempdir) }()
	tempfile := filepath.Join(tempdir, "value")
	cmd := exec.Command(editor, tempfile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(tempfile)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
