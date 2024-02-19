package main

import (
	"log"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/sio/pond/initramfs/cpio"
)

var pre = struct {
	Init   string
	Output string
}{
	Init:   os.Getenv("PRE_INIT"),
	Output: os.Getenv("PRE_OUTPUT"),
}

func main() {
	file, err := os.OpenFile(pre.Output, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	err = file.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}
	compressor, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		log.Fatal(err)
	}
	initramfs := cpio.New(compressor)
	err = initramfs.Copy(pre.Init, "init")
	if err != nil {
		log.Fatal(err)
	}
	err = initramfs.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = compressor.Close()
	if err != nil {
		log.Fatal(err)
	}
}
