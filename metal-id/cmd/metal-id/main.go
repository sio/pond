package main

import (
	"github.com/sio/pond/metal_id"

	"crypto"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	// Runtime checks
	checkOS()
	if os.Geteuid() != 0 {
		stderr("WARNING: When running as non-root some data sources will not be available.")
		stderr("WARNING: Hardware derived key will not match the one generated by root.")
	}

	// Parse CLI arguments
	verbose := flag.Bool("verbose", false, "print some information about fingerprint data (safe)")
	unsafe := flag.Bool("unsafe", false, "print non-obfuscated debug information (unsafe)")
	dest := flag.String("output", "", `private key destination path`)
	paranoid := flag.Bool("paranoid", false, "use extra paranoid data sources for fingerprinting")
	flag.Parse()
	var err error
	if len(*dest) > 0 {
		dir := filepath.Dir(*dest)
		if _, err = os.Stat(dir); os.IsNotExist(err) {
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
	var src = metal_id.Sources()
	if *paranoid {
		for name, datasource := range metal_id.SourcesParanoid() {
			src[name] = datasource
		}
	}
	var names = make([]string, len(src))
	var i int
	for name := range src {
		names[i] = name
		i++
	}
	sort.Strings(names)
	var hwid = metal_id.New()
	for _, name := range names {
		debug("Reading %s", name)
		data := src[name]
		for {
			chunk := data.Next()
			if data.Err() != nil {
				fail("Fetching data: %v", data.Err())
			}
			if chunk == nil {
				break
			}
			if len(chunk) == 0 {
				continue
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
	err = os.WriteFile(*dest+".pub", output, 0644)
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
