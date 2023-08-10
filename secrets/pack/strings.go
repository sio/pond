// A bespoke binary serialization format
package pack

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

// Serialize a slice of strings to an efficient binary representation
func Encode(s []string) ([]byte, error) {
	var err error
	var buf bytes.Buffer

	// First we write the number of elements in the slice
	if len(s) > utf8.MaxRune {
		return nil, fmt.Errorf("input too long: %d elements; max allowed: %d", len(s), utf8.MaxRune)
	}
	_, err = buf.Write(Uint(len(s)).Bytes())
	if err != nil {
		return nil, err
	}

	// Then size of each element
	for index, elem := range s {
		if len(elem) > utf8.MaxRune {
			return nil, fmt.Errorf(
				"element %d is too large: %d bytes; max allowed: %d",
				index,
				len(elem),
				utf8.MaxRune,
			)
		}
		_, err = buf.Write(Uint(len(elem)).Bytes())
		if err != nil {
			return nil, err
		}
	}

	// Then the elements themselves
	for _, elem := range s {
		_, err = buf.Write([]byte(elem))
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// Deserialize a slice of strings saved previously
//
// Even though serialization format allows for streaming data, we expect to
// receive the full input at once - for simplicity.
// Another implementation may be added to consume io.Reader.
func Decode(b []byte) ([]string, error) {

	// Read elements count and lengths
	header := utf8.UTFMax
	if header > len(b) {
		header = len(b)
	}
	var r []rune
	r = bytes.Runes(b[:header])
	if len(r) == 0 {
		return nil, fmt.Errorf("input is too short to deserialize")
	}
	header = utf8.UTFMax * int(r[0])
	if header == 0 { // an empty slice was decoded
		return []string{}, nil
	}
	if header > len(b) {
		header = len(b)
	}
	r = bytes.Runes(b[:header])
	if len(r)-1 < int(r[0]) {
		return nil, fmt.Errorf("input is too short: header advertizes %d elements, only %d lengths provided in first %d bytes", int(r[0]), len(r)-1, header)
	}
	r = r[:int(r[0])+1]

	// Skip serialization header
	var offset int
	for i := 0; i < len(r); i++ {
		offset += len(Uint(r[i]).Bytes())
	}

	// Read elements
	var s = make([]string, int(r[0]))
	for i := 0; i < len(s); i++ {
		size := int(r[i+1])
		if len(b) < offset+size {
			return nil, fmt.Errorf("unexpected end of input at element %d: input size %db, element size %db, current offset %db, previous elements %v", i, len(b), size, offset, s[:i])
		}
		s[i] = string(b[offset : offset+size])
		offset += size
	}
	if offset != len(b) {
		return nil, fmt.Errorf("input left over after deserialization: %d bytes, decoded elements %v", len(b)-offset, s)
	}
	return s, nil
}
