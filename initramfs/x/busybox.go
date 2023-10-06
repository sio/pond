// Proof of concept: create a minimal initramfs in our code
//
// This program assumes that statically built busybox is available at
// /bin/busybox (Debian package: busybox-static).
package main

import (
	"log"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/sio/pond/initramfs/cpio"
)

func main() {
	file, err := os.OpenFile(os.Args[1], os.O_RDWR|os.O_CREATE, 0644)
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
	err = initramfs.Copy("/bin/busybox", "bin/busybox")
	if err != nil {
		log.Fatal(err)
	}
	err = initramfs.Link("/bin/busybox", "bin/sh")
	if err != nil {
		log.Fatal(err)
	}
	err = initramfs.Link("/bin/busybox", "bin/reboot")
	if err != nil {
		log.Fatal(err)
	}
	err = initramfs.Link("/bin/busybox", "init")
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
