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
}{
	Init:   os.Getenv("PRE_INIT"),
	Output: os.Getenv("PRE_OUTPUT"),
	Exe: []string{
		"/bin/ls",
		"/bin/mount",
		"/bin/setsid",
		"/bin/sh",
		"/bin/find",
		"/bin/mkdir",
	},
}

func main() {
	file, err := os.OpenFile(config.Output, os.O_RDWR|os.O_CREATE, 0644)
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
	err = initramfs.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = compressor.Close()
	if err != nil {
		log.Fatal(err)
	}
}
