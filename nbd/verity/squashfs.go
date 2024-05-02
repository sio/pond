package verity

import (
	"encoding/binary"
	"fmt"
	"io"
)

func verityAfterSquashfs(partition io.ReaderAt) (v Verity, err error) {
	reader := &offsetReader{Reader: partition, Offset: 0}

	// Parse squashfs superblock
	var squashfs squashfsSuperblock
	err = binary.Read(reader, binary.LittleEndian, &squashfs)
	if err != nil {
		return v, fmt.Errorf("reading squashfs superblock: %w", err)
	}
	err = squashfs.Validate()
	if err != nil {
		return v, fmt.Errorf("squashfs: %w", err)
	}

	// Assume 4k alignment (default in mksquashfs)
	const align = 4096
	v.superblockOffset = int64(squashfs.BytesUsed) / align * align
	if squashfs.BytesUsed%align != 0 {
		v.superblockOffset += align
	}

	// Assume that verity superblock and hash tree immediately follow squashfs partition
	reader.Offset = v.superblockOffset
	err = binary.Read(reader, binary.LittleEndian, &v.veritySuperblock)
	if err != nil {
		return v, fmt.Errorf("reading verity superblock: %w", err)
	}
	err = v.validate()
	if err != nil {
		return v, fmt.Errorf("verity: %w", err)
	}
	return v, nil
}

// Squashfs superblock
// <https://dr-emann.github.io/squashfs/squashfs.html#_the_superblock>
type squashfsSuperblock struct {
	Magic        uint32
	InodeCount   uint32
	ModTime      uint32
	BlockSize    uint32
	FragCount    uint32
	Compressor   uint16
	BlockLog     uint16
	Flags        uint16
	IdCount      uint16
	VersionMajor uint16
	VersionMinor uint16
	RootInode    uint64
	BytesUsed    uint64
	IdOffset     uint64
	XattrOffset  uint64
	InodeOffset  uint64
	DirOffset    uint64
	FragOffset   uint64
	ExportOffset uint64
}

func (sb *squashfsSuperblock) Validate() error {
	if sb.Magic != 0x73717368 {
		return fmt.Errorf("invalid superblock magic: %#x", sb.Magic)
	}
	if sb.VersionMajor != 4 {
		return fmt.Errorf("invalid major version: %d (%#x)", sb.VersionMajor, sb.VersionMajor)
	}
	if sb.VersionMinor != 0 {
		return fmt.Errorf("invalid major version: %d (%#x)", sb.VersionMinor, sb.VersionMinor)
	}
	return nil
}
