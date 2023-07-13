package metal_id

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/sha512"
	"fmt"
	"hash"

	"golang.org/x/crypto/argon2"
)

const (
	idVersion         uint = 1
	idHeaderRepeat    int  = 16
	dataPointMinBytes int  = 8
	dataPointMinCount uint = 8 // 8 data points * 8 bytes minimum * 8 bit = 512 bit minimum entropy
)

// Initialize new MetalID
func New() *MetalID {
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
		hash: h,
	}
}

type MetalID struct {
	// Hasher is used to calculate hardware fingerprint
	hash hash.Hash

	// Number of data points which were fed to hasher
	count uint
}

// Feed data to fingerprint function
func (id *MetalID) Write(p []byte) (n int, err error) {
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
	seed := argon2.IDKey(fingerprint[16:], fingerprint[:16], 4, 256*1024, 2, ed25519.SeedSize)
	return ed25519.NewKeyFromSeed(seed), nil
}
