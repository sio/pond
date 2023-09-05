package tests

import (
	"github.com/sio/pond/lib/bytepack"
	"testing"

	"bytes"
	"crypto/rand"
	"fmt"
	"io"
)

func FuzzPacking(f *testing.F) {
	f.Add("helloworld!", 10)
	f.Add("Foo Bar Baz", 1)
	f.Add("\n", 100)
	f.Add("0", 1)
	f.Add("0", 0)
	f.Fuzz(func(t *testing.T, element string, count int) {
		var remove int
		const cutAfter = 2
		remove = len(element) - cutAfter
		if remove < 0 {
			remove = 0
		}
		if count < 0 {
			count *= -1
		}
		plain := make([][]byte, count)
		for i := 0; i < len(plain); i++ {
			var cut int
			if remove != 0 {
				cut = i % remove
			}
			plain[i] = []byte(element[:len(element)-cut])
		}

		var pack *bytepack.Bytepack
		pack, err := bytepack.Pack(plain)
		if err != nil {
			t.Fatalf("encoding: %v", err)
		}

		if pack.Size() != len(plain) {
			t.Fatalf("pack size changed: was %d, now %d elements", len(plain), pack.Size())
		}
		for i := 0; i < len(plain); i++ {
			want := string(plain[i])
			got := string(pack.Element(i))
			if got != want {
				t.Errorf("element %d changed: was %q, now %q", i, want, got)
			}
		}
		if testing.Verbose() {
			var total int
			for _, e := range plain {
				total += len(e)
			}
			var overhead int = 100
			if total != 0 {
				overhead = (len(pack.Blob()) - total) * 100 / total
			}
			t.Logf(
				"input: %d elements, total %d bytes; output: %d bytes; overhead %d bytes (%d%%)",
				len(plain),
				total,
				len(pack.Blob()),
				len(pack.Blob())-total,
				overhead,
			)
			if total < 100 {
				t.Logf("\ninput:   %x\nencoded: %x", plain, pack.Blob())
			}
		}
	})
}

func BenchmarkBytepack(b *testing.B) {
	tests := []struct {
		elements int
		size     int
	}{
		{10, 5},
		{10, 100},
		{10, 1000},
		{10, 10000},
		{100, 5},
		{100, 100},
		{100, 1000},
		{100, 10000},
		{1000, 5},
		{1000, 100},
		{1000, 1000},
		{1000, 10000},
	}
	for _, tt := range tests {
		b.Run(fmt.Sprintf("%dx%dB", tt.elements, tt.size), func(b *testing.B) {
			var data = make([][]byte, tt.elements)
			for i := 0; i < len(data); i++ {
				data[i] = make([]byte, tt.size)
				_, err := io.ReadFull(rand.Reader, data[i])
				if err != nil {
					b.Fatalf("rand: %v", err)
				}
			}
			for i := 0; i < b.N; i++ {
				pack, err := bytepack.Pack(data)
				if err != nil {
					b.Fatalf("pack: %v", err)
				}
				if pack.Size() != len(data) {
					b.Fatalf("size mismatch: %d!=%d", pack.Size(), len(data))
				}
				for el := 0; el < len(data); el++ {
					if !bytes.Equal(data[el], pack.Element(el)) {
						b.Fatalf("element %d mismatch:\n%x\n%x", el, data[el], pack.Element(el))
					}
				}
			}
		})
	}
}
