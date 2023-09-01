package master

import (
	"encoding/ascii85"
	"encoding/base64"
	"testing"

	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"sort"
)

// Ascii85 is more storage efficient 25% overhead vs 33% overhead for Base64,
// but it's noticeably slower.
//
// It's not worth switching from Base64 for secrets storage.
//
// See also: https://stackoverflow.com/a/1443240

func BenchmarkBase64(b *testing.B) {
	for _, size := range sizes() {
		var (
			plain   = make([]byte, size)
			decoded = make([]byte, size)
			encoded = make([]byte, base64.StdEncoding.EncodedLen(size))
			err     error
		)
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err = io.ReadFull(rand.Reader, plain)
				if err != nil {
					b.Fatal(err)
				}
				base64.StdEncoding.Encode(encoded, plain)
				_, err = base64.StdEncoding.Decode(decoded, encoded)
				if err != nil {
					b.Fatal(err)
				}
				if !bytes.Equal(plain, decoded) {
					b.Fatal("decoded data does not match plain input")
				}
			}
		})
	}
}

func BenchmarkAscii85(b *testing.B) {
	for _, size := range sizes() {
		var (
			plain   = make([]byte, size)
			encoded = make([]byte, ascii85.MaxEncodedLen(size))
			decoded = make([]byte, len(encoded))
			err     error
		)
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				encoded = encoded[:cap(encoded)]
				_, err = io.ReadFull(rand.Reader, plain)
				if err != nil {
					b.Fatal(err)
				}
				n := ascii85.Encode(encoded, plain)
				encoded = encoded[:n]

				decoded = decoded[:cap(decoded)]
				n, _, err = ascii85.Decode(decoded, encoded, true)
				decoded = decoded[:n]
				if err != nil {
					b.Fatal(err)
				}
				if !bytes.Equal(plain, decoded) {
					if size < 1000 {
						b.Logf("\nwant: %x\n got: %x", plain, decoded)
					}
					b.Fatalf("decoded data does not match plain input")
				}
			}
		})
	}
}

func sizes() []int {
	var (
		size  int
		sizes []int
	)
	for _, base := range []float64{2, 10} {
		for _, step := range []float64{1, 2, 3, 4, 5, 6} {
			if base == 2 {
				size = int(math.Pow(base, step*10/3)) - 1
			} else {
				size = int(math.Pow(base, step)) + 1
			}
			sizes = append(sizes, size)
		}
	}
	sort.Ints(sizes)
	return sizes
}
