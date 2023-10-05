package cpio

import (
	"testing"

	"bytes"
)

var headerTests = []struct {
	header   Header
	path     string
	expected string
}{
	{
		Header{},
		"hello",
		"07070100000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000600000000hello\x00",
	},
	{
		Header{inode: 0x6334, mode: 040755, nlink: 3, mtime: 1696098165, maj: 0xFE, min: 0x02},
		"conf",
		"07070100006334000041ed0000000000000000000000036518677500000000000000fe0000000200000000000000000000000500000000conf\x00\x00",
	},
	{
		Header{uid: 10, gid: 0xFF, rmaj: 0x10, rmin: 0xABCD, checksum: 0},
		"using/unused/fields/to/silence/staticcheck",
		"07070100000000000000000000000a000000ff0000000100000000000000000000000000000000000000100000abcd0000002b00000000using/unused/fields/to/silence/staticcheck\x00\x00\x00\x00",
	},
}

func TestHeaderSerialization(t *testing.T) {
	var buf = bytes.NewBuffer(make([]byte, 4096))
	for index, tt := range headerTests {
		buf.Reset()
		err := tt.header.Write(buf, tt.path)
		if err != nil {
			t.Errorf("sample %d: %v", index, err)
			continue
		}
		if buf.String() != tt.expected {
			t.Errorf(
				"incorrect serialization (sample %d):\nwant: %s (%d bytes)\n got: %s (%d bytes)",
				index,
				tt.expected,
				len(tt.expected),
				buf.String(),
				buf.Len(),
			)
		}
	}
}

func BenchmarkHeaderSerialization(b *testing.B) {
	var buf = bytes.NewBuffer(make([]byte, 4096))
	for i := 0; i < b.N; i++ {
		index := i % len(headerTests)
		tt := headerTests[index]
		buf.Reset()
		err := tt.header.Write(buf, tt.path)
		if err != nil {
			b.Fatalf("sample %d: %v", index, err)
			continue
		}
		if buf.String() != tt.expected {
			b.Fatalf(
				"incorrect serialization (sample %d):\nwant: %s (%d bytes)\n got: %s (%d bytes)",
				index,
				tt.expected,
				len(tt.expected),
				buf.String(),
				buf.Len(),
			)
		}
	}
}
