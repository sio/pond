package main

import (
	"metal_id"

	"crypto"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	checkOS()

	// Parse CLI arguments
	verbose := flag.Bool("v", false, "print some information about fingerprint data (safe)")
	unsafe := flag.Bool("debug-unsafe", false, "do not obfuscate fingerprint data (unsafe)")
	dest := flag.String("f", "", `private key destination path`)
	flag.Parse()
	if len(*dest) > 0 {
		dir := filepath.Dir(*dest)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fail("Destination directory does not exist: %s", dir)
		}
	}
	if *unsafe {
		*verbose = true
	}
	debug := func(format string, a ...any) {
		if !(*verbose) {
			return
		}
		stderr(format, a...)
	}

	// Derive key from hardware fingerprint
	var hwid = metal_id.New()
	var err error
	for _, data := range metal_id.Sources() {
		debug("Reading %s", data.Name)
		for {
			var chunk []byte
			chunk = data.Next()
			if len(chunk) == 0 {
				break
			}
			debug("  %s", previewSeed(chunk, *unsafe))
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

	// Print public key to standard output
	var output []byte
	output, err = metal_id.EncodePublicKey(key.Public())
	if err != nil {
		fail("Failed to serialize public key: %v", err)
	}
	fmt.Println(string(output))

	// Save keys to file system
	if len(*dest) == 0 {
		return
	}
	err = os.WriteFile(*dest+".pub", output, 0600)
	if err != nil {
		fail("Failed to save public key: %v", err)
	}
	output, err = metal_id.EncodePrivateKey(key)
	if err != nil {
		fail("Failed to serialize private key: %v", err)
	}
	err = os.WriteFile(*dest, output, 0600)
	if err != nil {
		fail("Failed to save private key: %v", err)
	}
}
