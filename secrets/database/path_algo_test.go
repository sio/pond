// Selecting KDF algorithm for path hashing
//
// We need a hash function for writing non-reversible paths to database.
// Paths have to be non-reversible to complicate attacker's life in case of a
// private key leak.
// Simple cryptographic hashes are vulnerable to dictionary attacks since we do
// not expect high entropy paths.
//
// Our KDF needs to be modern enough to provide some level of protection
// against brute forcing (even though paths are not considered sensitive),
// but fast enough not to overload our server while serving legitimate requests.

package database

import (
	"testing"

	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
	"hash"
	"io"
)

const (
	keyBytes  = 32
	ikmBytes  = 32
	saltBytes = 32
)

func BenchmarkKDFArgon2(b *testing.B) {
	benchmarks := []struct {
		mem     uint32
		time    uint32
		threads uint8
	}{
		{64, 1, 4},
		{64, 1, 2},
		{64, 2, 4},
		{64, 2, 2},
		{64, 4, 2},
		{64, 4, 4},
		{128, 1, 4},
		{128, 1, 2},
		{128, 2, 4},
		{128, 2, 2},
		{128, 4, 2},
		{128, 4, 4},
		{256, 1, 4},
		{256, 1, 2},
		{256, 2, 4},
		{256, 2, 2},
		{256, 4, 2},
		{256, 4, 4},
	}
	var err error
	var input = make([]byte, ikmBytes+saltBytes)
	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("%dMB/time%d/threads%d", bm.mem, bm.time, bm.threads), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err = io.ReadFull(rand.Reader, input)
				if err != nil {
					b.Fatal(err)
				}
				_ = argon2.IDKey(input[saltBytes:], input[:saltBytes], bm.time, bm.mem*1024, bm.threads, keyBytes)
			}
		})
	}
}

func BenchmarkKDFBcrypt(b *testing.B) {
	benchmarks := []struct {
		cost int
	}{
		{0},
		{4},
		{8},
		{10},
		{16},
		// higher costs are extremely slow
	}
	var err error
	var input = make([]byte, ikmBytes)
	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("cost%d", bm.cost), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err = io.ReadFull(rand.Reader, input)
				if err != nil {
					b.Fatal(err)
				}
				_, err = bcrypt.GenerateFromPassword(input, bm.cost)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkKDFScrypt(b *testing.B) {
	benchmarks := []struct {
		N int
		r int
		p int
	}{
		{32768, 8, 1}, // recommended for interactive logins
		{32768, 8, 1 * 2},
		{32768, 8, 1 * 4},
		{32768, 8, 1 * 8},
		{32768, 8 * 2, 1},
		{32768, 8 * 4, 1},
		{32768, 8 * 8, 1},
		{32768 * 2, 8, 1},
		{32768 * 2, 8, 1 * 2},
		{32768 * 2, 8, 1 * 4},
		{32768 * 2, 8, 1 * 8},
		{32768 * 2, 8 * 2, 1},
		{32768 * 2, 8 * 4, 1},
		{32768 * 2, 8 * 8, 1},
		{32768 * 4, 8, 1},
		{32768 * 4, 8, 1 * 2},
		{32768 * 4, 8, 1 * 4},
		{32768 * 4, 8, 1 * 8},
		{32768 * 4, 8 * 2, 1},
		{32768 * 4, 8 * 4, 1},
		{32768 * 4, 8 * 8, 1},
		{32768 * 8, 8, 1},
		{32768 * 8, 8, 1 * 2},
		{32768 * 8, 8, 1 * 4},
		{32768 * 8, 8, 1 * 8},
		{32768 * 8, 8 * 2, 1},
		{32768 * 8, 8 * 4, 1},
		{32768 * 8, 8 * 8, 1},
	}
	var err error
	var input = make([]byte, ikmBytes+saltBytes)
	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("N%d/r%d/p%d", bm.N, bm.r, bm.p), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err = io.ReadFull(rand.Reader, input)
				if err != nil {
					b.Fatal(err)
				}
				_, err = scrypt.Key(input[saltBytes:], input[:saltBytes], bm.N, bm.r, bm.p, keyBytes)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkKDFpbkdf2(b *testing.B) {
	var hash = map[string]func() hash.Hash{
		"sha256": sha256.New,
		"sha512": sha512.New,
	}
	benchmarks := []struct {
		iter int
		hash string
	}{
		{1024, "sha256"},
		{1024 * 2, "sha256"},
		{1024 * 4, "sha256"},
		{1024 * 16, "sha256"},
		{1024, "sha512"},
		{1024 * 2, "sha512"},
		{1024 * 4, "sha512"},
		{1024 * 16, "sha512"},
	}
	var err error
	var input = make([]byte, ikmBytes+saltBytes)
	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("%s/iter%d", bm.hash, bm.iter), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err = io.ReadFull(rand.Reader, input)
				if err != nil {
					b.Fatal(err)
				}
				_ = pbkdf2.Key(input[saltBytes:], input[:saltBytes], bm.iter, keyBytes, hash[bm.hash])
			}
		})
	}
}
