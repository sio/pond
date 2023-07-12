package main

import (
	"metal_id"

	"crypto"
	"fmt"
	"os"
	"strings"
)

func main() {
	var hwid = metal_id.New()

	var err error
	var data metal_id.DataSource
	for _, data = range metal_id.Sources() {
		for {
			var chunk []byte
			chunk = data.Next()
			if len(chunk) == 0 {
				break
			}
			_, err = hwid.Write(chunk)
			if err != nil {
				fail("Failed to add data to fingerprint: %v", err)
			}
		}
	}

	var key crypto.Signer
	key, err = hwid.Key()
	if err != nil {
		fail("Failed to generate machine key: %v", err)
	}

	var output []byte
	output, err = metal_id.EncodePublicKey(key.Public())
	if err != nil {
		fail("Failed to encode public key: %v", err)
	}
	fmt.Println(string(output))
}

func fail(format string, a ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	_, err := fmt.Fprintf(os.Stderr, format, a...)
	if err != nil {
		panic("failed to write to stderr: " + fmt.Sprint(err))
	}
	os.Exit(1)
}
