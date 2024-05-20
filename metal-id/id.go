package metal_id

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/sha512"
	"fmt"
	"hash"
	"sort"

	"golang.org/x/crypto/argon2"
)

const (
	idVersion         uint = 1
	idHeaderRepeat    int  = 16
	dataPointMinBytes int  = 8
	dataPointMinCount uint = 8 // 8 data points * 8 bytes minimum * 8 bit = 512 bit minimum entropy
)

// Initialize new MetalID
func New(debug func(f string, a ...any), sensitive func(d []byte) string) *MetalID {
	h := sha512.New()
	_, err := h.Write(
		bytes.Repeat(
			[]byte(fmt.Sprintf("METAL-ID VERSION %d", idVersion)),
			idHeaderRepeat,
		),
	)
	if err != nil {
		panic("failed to initialize MetalID hash with version header")
	}
	return &MetalID{
		hash:          h,
		debugFunc:     debug,
		sensitiveFunc: sensitive,
	}
}

type MetalID struct {
	// Hasher is used to calculate hardware fingerprint
	hash hash.Hash

	// Number of data points which were fed to hasher
	count uint

	// Debug logger
	debugFunc func(format string, args ...any)

	// Format sensitive values for viewing
	sensitiveFunc func(d []byte) string
}

// TODO: Use TPM (if present) to generate deterministic machine identity key instead of our bespoke algorithm (see below)
// Relevant links:
//  * https://ericchiang.github.io/post/tpm-keys/
//  * https://dev.to/nandhithakamal/tpm-part-1-4emf
//  * https://github.com/tpm2dev/tpm.dev.tutorials/blob/master/Intro/README.md

// Feed data to fingerprint function
func (id *MetalID) write(p []byte) (n int, err error) {
	n, err = id.hash.Write(p)
	if n >= dataPointMinBytes && err == nil {
		id.count++
	}
	return n, err
}

// Derive private key from hardware fingerprint
func (id *MetalID) Key() (crypto.Signer, error) {
	if id.count < dataPointMinCount {
		return nil, fmt.Errorf("not enough data points (want %d, have %d)", dataPointMinCount, id.count)
	}
	if id.hash.Size() < 16+16 {
		return nil, fmt.Errorf("hash sum is too short for safe key derivation")
	}
	fingerprint := id.hash.Sum(nil)

	// Argon2 parameter selection logic:
	//   time=4
	//     We are not in a hurry. This key will typically be generated once per boot.
	//   memory=256MB
	//     We are not targeting embedded devices, and pretty much any server
	//     should have 256MB free early after boot.
	//   threads=2
	//     We target cheap old hardware. Extremely multicore CPUs are not guaranteed.
	salt := sha512.Sum512(fingerprint)
	seed := argon2.IDKey(fingerprint, salt[:], 4, 256*1024, 2, ed25519.SeedSize)
	return ed25519.NewKeyFromSeed(seed), nil
}

// Fetch datasources and generate hardware fingerprint
func (id *MetalID) Fetch(src map[string]DataSource) error {
	var names = make([]string, len(src))
	var i int
	for name := range src {
		names[i] = name
		i++
	}
	sort.Strings(names)
	for _, name := range names {
		id.debug("Reading %s", name)
		data := src[name]
		for {
			chunk := data.Next()
			if data.Err() != nil {
				return fmt.Errorf("reading %s: %w", name, data.Err())
			}
			if chunk == nil {
				break
			}
			if len(chunk) == 0 {
				continue
			}
			id.debug("  %s", id.sensitive(chunk))
			_, err := id.write(chunk)
			if err != nil {
				return fmt.Errorf("adding %s: %w", name, err)
			}
		}
	}
	return nil
}

// Print debug information
func (id *MetalID) debug(format string, args ...any) {
	if id.debugFunc == nil {
		return
	}
	id.debugFunc(format, args...)
}

// Format sensitive information for viewing
func (id *MetalID) sensitive(d []byte) string {
	if id.sensitiveFunc == nil {
		return "(sensitive)"
	}
	return id.sensitiveFunc(d)
}
