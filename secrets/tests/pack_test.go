package tests

import (
	"testing"

	"secrets/pack"
)

func FuzzPacking(f *testing.F) {
	f.Add("helloworld!", 10)
	f.Add("Foo Bar Baz", 1)
	f.Add("\n", 100)
	f.Add("0", 1)
	f.Add("0", 0)
	f.Fuzz(func(t *testing.T, element string, count int) {
		var remove int
		const cutAfter = 5
		remove = len(element) - cutAfter
		if remove < 0 {
			remove = 0
		}
		if count < 0 {
			count *= -1
		}
		plain := make([]string, count)
		for i := 0; i < len(plain); i++ {
			var cut int
			if remove != 0 {
				cut = i % remove
			}
			plain[i] = element[:len(element)-cut]
		}
		encoded, err := pack.Encode(plain)
		if err != nil {
			t.Fatalf("encoding: %v", err)
		}
		decoded, err := pack.Decode(encoded)
		if err != nil {
			t.Fatalf("decoding: %v", err)
		}
		if len(decoded) != len(plain) {
			t.Fatalf("slice size changed: was %d, now %d elements", len(plain), len(decoded))
		}
		for i := 0; i < len(plain); i++ {
			if plain[i] != decoded[i] {
				t.Errorf("element %d changed: was %q, now %q", i, plain[i], decoded[i])
			}
		}
		if testing.Verbose() {
			var total int
			for _, e := range plain {
				total += len(e)
			}
			var overhead int = -1
			if total != 0 {
				overhead = (len(encoded)-total)*100/total
			}
			t.Logf(
				"input: %d strings, total %d bytes; output: %d bytes; overhead %d bytes (%d%%)",
				len(plain),
				total,
				len(encoded),
				len(encoded)-total,
				overhead,
			)
			if total < 100 {
				t.Logf("\ninput:   %q\nencoded: %x", plain, encoded)
			}
		}
	})
}
