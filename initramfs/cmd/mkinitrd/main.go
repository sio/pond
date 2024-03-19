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
	Init    string
	Output  string
	Exe     []string
	Lib     []string
	Copy    map[string]string // destination: source
	Kmod    []string
	KmodDir string
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
		"/bin/lspci",
		"/bin/ip",
		"/bin/ping",
		"/sbin/dhclient",
		"/sbin/modprobe",
		"/bin/curl",
		"/bin/strace",
	},
	Lib: []string{
		"libnss_files.so.2",
		"libnss_dns.so.2",
	},
	Copy: map[string]string{
		// These three modules form a dependency tree:
		//    ata_generic -> libata -> scsi_mod
		// Try deleting any of the dependencies and see what happens in `make demo`
		//"/lib/modules/5.10.0-19-amd64/kernel/drivers/ata/ata_generic.ko": "",
		//"/lib/modules/5.10.0-19-amd64/kernel/drivers/ata/libata.ko":      "",
		//"/lib/modules/5.10.0-19-amd64/kernel/drivers/scsi/scsi_mod.ko":   "",
		"/etc/motd": "",
	},
	Kmod: []string{
		"e1000",  // default QEMU network card
		"8139cp", // another QEMU emulated NIC, Realtek 8139
		"ata_generic",
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
	// Build filesystem tree in memory
	fs := make(map[string]string) // dest: src
	cp := func(src, dest string) {
		if src == "" {
			src = dest
		}
		if dest != "" && dest[0] == '/' {
			dest = dest[1:]
		}
		fs[dest] = src
	}

	cp(config.Init, "init")
	for _, exe := range config.Exe {
		cp(exe, exe)
		deps, err := ldd.Depends(exe)
		if err != nil {
			return fail(err)
		}
		for _, lib := range deps {
			cp(lib, lib)
		}
	}
	for _, lib := range config.Lib {
		lib, err := ldd.Library(lib, nil)
		if err != nil {
			return fail(err)
		}
		deps, err := ldd.Depends(lib)
		if err != nil {
			return fail(err)
		}
		cp(lib, lib)
		for _, lib = range deps {
			cp(lib, lib)
		}
	}
	modules, err := findKernelModules(config.KmodDir, config.Kmod)
	if err != nil {
		return fail(err)
	}
	for _, module := range modules {
		cp(module.src, module.dest)
	}
	for dest, src := range config.Copy {
		cp(src, dest)
	}

	// Actually add files to archive
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
	for dest, src := range fs {
		fmt.Printf("%s -> %s\n", src, dest)
		err := initramfs.Copy(src, dest)
		if err != nil {
			fail(err)
		}
	}
	return rc
}
