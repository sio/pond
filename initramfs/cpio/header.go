package cpio

import (
	"encoding/binary"
	"encoding/hex"
	"io"
)

// cpio header in new ASCII format
type Header struct {
	magic    [3]byte
	inode    uint32
	mode     uint32
	uid      uint32
	gid      uint32
	nlink    uint32
	mtime    uint32
	filesize uint32
	maj      uint32
	min      uint32
	rmaj     uint32
	rmin     uint32
	namesize uint32
	checksum uint32
}

func (h Header) Write(w io.Writer, path string) error {
	const (
		align        = 4
		trailingNull = 1
	)
	h.magic = magicNewAsciiNoChecksum
	h.namesize = uint32(len(path) + trailingNull)
	if h.nlink == 0 {
		h.nlink = 1
	}
	err := binary.Write(hex.NewEncoder(w), binary.BigEndian, h)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(path))
	if err != nil {
		return err
	}
	var zero [align + trailingNull]byte
	padding := (len(h.magic)*2 + len(path) + trailingNull) % align
	if padding != 0 {
		padding = align - padding
	}
	_, err = w.Write(zero[:trailingNull+padding])
	if err != nil {
		return err
	}
	return nil
}
