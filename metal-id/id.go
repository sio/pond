package metal_id

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"hash"
)

const (
	idVersion         uint = 1
	idHeaderRepeat    int  = 16
	dataPointMinBytes int  = 8
	dataPointMinCount uint = 4 // 4 data points * 8 bytes minimum * 8 bit = 256 bit minimum entropy
)

// Initialize new MetalID
func New() *MetalID {
	h := sha256.New()
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

	// Prevent ed25519.NewKeyFromSeed from panicking
	if id.hash.Size() < ed25519.SeedSize {
		return nil, fmt.Errorf("hash sum is too short for key derivation (want %d, have %d bytes)", ed25519.SeedSize, id.hash.Size())
	}

	return ed25519.NewKeyFromSeed(id.hash.Sum(nil)[:ed25519.SeedSize]), nil
}
