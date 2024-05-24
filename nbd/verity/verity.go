// Quick data integrity verification using dm-verity hash tree.
// This is not intended to be a full verity implementation.
package verity

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
)

// Parse verity hash tree appended to data partition.
// Currently only squashfs partitions are supported.
//
// Does not capture the provided reader.
func Open(r io.ReaderAt) (Verity, error) {
	return verityAfterSquashfs(r)
}

type Verity struct {
	veritySuperblock
	superblockOffset int64
	leafHashOffset   int64
}

// Verify integrity of data region described by provided offset and size.
//
// This check guards only against accidental data corruption:
// only leaf hashes of verity tree are being calculated and compared.
// Use cryptsetup tools to get full integrity guarantees provided by dm-verity.
func (t *Verity) Verify(r io.ReaderAt, offset int64, size int) error {
	hash := t.hash()
	if t.leafHashOffset < t.superblockOffset {
		t.findFirstLeafHash(hash)
	}
	for block := int(offset / int64(t.DataBlockSize)); size > 0 && uint64(block) < t.DataBlockCount; block++ {
		err := t.verifyBlock(r, hash, block)
		if err != nil {
			return err
		}
		size -= int(t.DataBlockSize)
	}
	return nil
}

// Verify integrity of data block with given index
func (t *Verity) verifyBlock(r io.ReaderAt, hash hash.Hash, index int) error {
	// Expected data block hash
	want := make([]byte, hash.Size())
	n, err := r.ReadAt(want, t.leafHashOffset+int64(index)*int64(hash.Size()))
	if err != nil {
		return fmt.Errorf("reading verity hash: %w", err)
	}
	if n != len(want) {
		return fmt.Errorf("reading verity hash: short read")
	}

	// Actual data block hash
	hash.Reset()
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

func (t *Verity) findFirstLeafHash(hash hash.Hash) {
	hashesPerBlock := int64(t.HashBlockSize) / int64(hash.Size())
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

// Validate verity superblock
func (sb *veritySuperblock) validate() error {
	if string(sb.Magic[:]) != "verity\x00\x00" {
		return fmt.Errorf("invalid superblock magic: %q (%#x)", string(sb.Magic[:]), sb.Magic)
	}
	if sb.Version != 1 {
		return fmt.Errorf("invalid superblock version: %d (%#x)", sb.Version, sb.Version)
	}
	if sb.Type != 1 {
		return fmt.Errorf("unsupported superblock type: %d (%#x)", sb.Type, sb.Type)
	}
	var hashSize uint16
	switch string(sb.Algorithm[:bytes.IndexByte(sb.Algorithm[:], 0)]) {
	case "sha256":
		hashSize = sha256.Size
	default:
		return fmt.Errorf("unsupported hash algorithm: %s (%#x)", string(sb.Algorithm[:]), sb.Algorithm)
	}
	if sb.SaltSize < hashSize {
		return fmt.Errorf("salt too short: %d bit", int(sb.SaltSize)*8)
	}
	var zero [256]byte
	if sb.Salt == zero {
		return fmt.Errorf("empty salt in superblock: %#x", sb.Salt)
	}
	return nil
}

func (sb *veritySuperblock) hash() hash.Hash {
	const prefix = 6 // number of bytes that uniquely identify each supported algorithm
	switch string(sb.Algorithm[:prefix]) {
	case "sha256":
		return sha256.New()
	default:
		panic("attempting to initialize hash function before superblock validation")
	}
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
