package verity

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
)

type verityTree struct {
	veritySuperblock
	superblockOffset int64
	leafHashOffset   int64
}

// Verify integrity of data block with given index
func (t *verityTree) Verify(r io.ReaderAt, index int) error {
	// Housekeeping first
	if t.leafHashOffset < t.superblockOffset {
		t.findFirstLeafHash()
	}

	// Expected data block hash
	want := make([]byte, sha256.Size)
	n, err := r.ReadAt(want, t.leafHashOffset+int64(index)*int64(t.HashBlockSize))
	if err != nil {
		return fmt.Errorf("reading verity hash: %w", err)
	}
	if n != len(want) {
		return fmt.Errorf("reading verity hash: short read")
	}

	// Actual data block hash
	hash := sha256.New()
	_, err = hash.Write(t.Salt[:int(t.SaltSize)])
	if err != nil {
		return err
	}
	reader := &offsetReader{Reader: r, Offset: int64(index) * int64(t.DataBlockSize)}
	_, err = io.CopyN(hash, reader, int64(t.DataBlockSize))
	if err != nil {
		return err
	}
	got := hash.Sum(nil)
	if !bytes.Equal(got, want) {
		return fmt.Errorf("hash mismatch for block %d: want %x, got %x", index, want, got)
	}
	return nil
}

func (t *verityTree) findFirstLeafHash() {
	hashesPerBlock := int64(t.HashBlockSize) / sha256.Size
	var layer []int64
	var i int
	blocks := int64(t.DataBlockCount)
	for blocks > 1 {
		layer = append(layer, blocks/hashesPerBlock)
		if blocks%hashesPerBlock != 0 {
			layer[i]++
		}
		blocks = layer[i]
		i++
	}
	var offset int64
	for i = 1; i < len(layer); i++ {
		offset += layer[i] * int64(t.HashBlockSize)
	}
	t.leafHashOffset = t.superblockOffset + int64(t.HashBlockSize) + offset
}

// Verity superblock
// <https://gitlab.com/cryptsetup/cryptsetup/-/wikis/DMVerity#verity-superblock-format>
type veritySuperblock struct {
	Magic          [8]byte
	Version        uint32
	Type           uint32
	UUID           [16]byte
	Algorithm      [32]byte
	DataBlockSize  uint32
	HashBlockSize  uint32
	DataBlockCount uint64
	SaltSize       uint16
	_              [6]byte
	Salt           [256]byte
	_              [128]byte
}

func (sb *veritySuperblock) Validate() error {
	if string(sb.Magic[:]) != "verity\x00\x00" {
		return fmt.Errorf("invalid superblock magic: %q (%#x)", string(sb.Magic[:]), sb.Magic)
	}
	if sb.Version != 1 {
		return fmt.Errorf("invalid superblock version: %d (%#x)", sb.Version, sb.Version)
	}
	if sb.Type != 1 {
		return fmt.Errorf("unsupported superblock type: %d (%#x)", sb.Type, sb.Type)
	}
	if string(sb.Algorithm[:bytes.IndexByte(sb.Algorithm[:], 0)]) != "sha256" {
		return fmt.Errorf("unsupported hash algorithm: %s (%#x)", string(sb.Algorithm[:]), sb.Algorithm)
	}
	if sb.SaltSize < sha256.Size {
		return fmt.Errorf("salt too short: %d bit", int(sb.SaltSize)*8)
	}
	var zero [256]byte
	if sb.Salt == zero {
		return fmt.Errorf("empty salt in superblock: %#x", sb.Salt)
	}
	return nil
}

type offsetReader struct {
	Reader io.ReaderAt
	Offset int64
}

func (r *offsetReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.ReadAt(p, r.Offset)
	r.Offset += int64(n)
	return n, err
}
