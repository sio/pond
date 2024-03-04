package main

import (
	"fmt"
	"io"
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

var rc int

func fail(f any, v ...any) int {
	s, ok := f.(string)
	if !ok || len(v) == 0 {
		log.Print(f)
	} else {
		log.Printf(s, v...)
	}
	rc = 11
	return rc
}

func _close(f io.Closer) {
	err := f.Close()
	if err != nil && rc == 0 {
		fail(err)
	}
}

func main() {
	run()
	os.Exit(rc)
}

func run() int {
	file, err := os.OpenFile(config.Output, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fail(err)
	}
	defer _close(file)

	err = file.Truncate(0)
	if err != nil {
		return fail(err)
	}

	compressor, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return fail(err)
	}
	defer _close(compressor)

	initramfs := cpio.New(compressor)
	defer _close(initramfs)
	cp := func(src, dest string) bool {
		fmt.Printf("%s -> %s\n", src, dest)
		err = initramfs.Copy(src, dest)
		if err != nil {
			fail(err)
			return false
		}
		return true
	}

	cp(config.Init, "init")
	for _, exe := range config.Exe {
		if !cp(exe, exe) {
			return rc
		}
		deps, err := ldd.Depends(exe)
		if err != nil {
			return fail(err)
		}
		for _, lib := range deps {
			if !cp(lib, lib) {
				return rc
			}
		}
	}
	for dest, src := range config.Copy {
		if src == "" {
			src = dest
		}
		if len(dest) > 0 && dest[0] == '/' {
			dest = dest[1:]
		}
		if !cp(src, dest) {
			return rc
		}
	}
	return rc
}
