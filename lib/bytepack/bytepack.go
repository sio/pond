// A bespoke binary serialization format
// for storing schema-less sequences of bytestrings
package bytepack

import (
	"bytes"
	"errors"
	"fmt"
	"unicode/utf8"
)

// Serialize a sequence of bytestrings to a single blob
func Pack(input [][]byte) (*Bytepack, error) {
	// Prepare output buffer
	var ceil int
	ceil += utf8.UTFMax
	for i := 0; i < len(input); i++ {
		ceil += utf8.UTFMax
		ceil += len(input[i])
	}
	var buf = make([]byte, ceil)
	var cur int

	// First we declare the number of elements
	cur += Uint(len(input)).Encode(buf[cur:])

	// Then provide length of each element
	for _, elem := range input {
		cur += Uint(len(elem)).Encode(buf[cur:])
	}

	// Then elements themselves
	for _, elem := range input {
		cur += copy(buf[cur:], elem)
	}

	// Wrap result into Bytepack struct
	pack, err := Wrap(buf[:cur])
	if err != nil {
		return nil, fmt.Errorf("serialization failed: %w", err)
	}
	return pack, nil
}

// Wrap encoded bytepack to access elements without making a copy
func Wrap(encoded []byte) (*Bytepack, error) {
	size, offset, err := decodeHead(encoded)
	if err != nil {
		return nil, err
	}
	return &Bytepack{
		data:   encoded,
		size:   size,
		offset: offset,
	}, nil
}

func decodeHead(encoded []byte) (size, offset []int, err error) {
	// First we parse total number of elements
	head := utf8.UTFMax
	if head > len(encoded) {
		head = len(encoded)
	}
	if head == 0 {
		return nil, nil, nil
	}
	var r []rune
	r = bytes.Runes(encoded[:head])
	if len(r) == 0 {
		return nil, nil, errors.New("encoded data is too short")
	}

	// Then all element lengths
	head += utf8.UTFMax * int(r[0])
	if head > len(encoded) {
		head = len(encoded)
	}
	if r[0] == 0 {
		return nil, nil, nil
	}
	r = bytes.Runes(encoded[:head])
	if len(r) < int(r[0])+1 {
		return nil, nil, fmt.Errorf("encoded data advertizes %d elements, but only %d sizes are provided", int(r[0]), len(r)-1)
	}
	r = r[:int(r[0])+1]

	// Calculate exact length of the header
	head = 0
	for i := 0; i < len(r); i++ {
		head += Uint(r[i]).Size()
	}

	size = make([]int, int(r[0]))
	offset = make([]int, int(r[0]))
	for i := 0; i < len(r)-1; i++ {
		size[i] = int(r[i+1])
		if i == 0 {
			offset[i] = head
		} else {
			offset[i] = offset[i-1] + size[i-1]
		}
	}
	got := offset[len(offset)-1] + size[len(size)-1]
	if got != len(encoded) {
		return nil, nil, fmt.Errorf("calculated blob size (%dB) does not match encoded size (%dB)", got, len(encoded))
	}
	return size, offset, nil
}

// Access encoded values without making a copy
type Bytepack struct {
	data   []byte
	offset []int
	size   []int
}

// Number of elements in this pack
func (p *Bytepack) Size() int {
	return len(p.offset)
}

// Encoded blob
func (p *Bytepack) Blob() []byte {
	return p.data
}

// Return pack element by index. Will panic on out of bounds access
func (p *Bytepack) Element(i int) []byte {
	return p.data[p.offset[i] : p.offset[i]+p.size[i]]
}

// Return all pack elements
func (p *Bytepack) Unpack() [][]byte {
	out := make([][]byte, len(p.offset))
	for i := 0; i < len(out); i++ {
		out[i] = p.Element(i)
	}
	return out
}
