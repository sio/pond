package main

import (
	"fmt"
	"log"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/sio/pond/initramfs/cpio"
	"github.com/sio/pond/initramfs/ldd"
)

var config = struct {
	Init   string
	Output string
	Exe    []string
	Copy   map[string]string // destination: source
}{
	Init:   os.Getenv("INIT"),
	Output: os.Getenv("INITRD"),
	Exe: []string{
		"/bin/ls",
		"/bin/mount",
		"/bin/setsid",
		"/bin/sh",
		"/bin/find",
		"/bin/mkdir",
		"/bin/cat",
		"/bin/sort",
		"/sbin/modprobe",
	},
	Copy: map[string]string{
		// These three modules form a dependency tree:
		//    ata_generic -> libata -> scsi_mod
		// Try deleting any of the dependencies and see what happens in `make demo`
		"/lib/modules/5.10.0-19-amd64/kernel/drivers/ata/ata_generic.ko": "",
		"/lib/modules/5.10.0-19-amd64/kernel/drivers/ata/libata.ko":      "",
		"/lib/modules/5.10.0-19-amd64/kernel/drivers/scsi/scsi_mod.ko":   "",
	},
}

func main() {
	file, err := os.OpenFile(config.Output, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	err = file.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}
	compressor, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		log.Fatal(err)
	}
	initramfs := cpio.New(compressor)
	cp := func(src, dest string) {
		fmt.Printf("%s -> %s\n", src, dest)
		err = initramfs.Copy(src, dest)
		if err != nil {
			log.Fatal(err)
		}
	}

	cp(config.Init, "init")
	for _, exe := range config.Exe {
		cp(exe, exe)
		deps, err := ldd.Depends(exe)
		if err != nil {
			log.Fatal(err)
		}
		for _, lib := range deps {
			cp(lib, lib)
		}
	}
	for dest, src := range config.Copy {
		if src == "" {
			src = dest
		}
		if len(dest) > 0 && dest[0] == '/' {
			dest = dest[1:]
		}
		cp(src, dest)
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
