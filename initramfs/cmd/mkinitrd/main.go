package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/sio/pond/initramfs/cpio"
	"github.com/sio/pond/initramfs/ldd"
)

var config = struct {
	Init    string
	Output  string
	Exe     []string
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
	},
	Copy: map[string]string{
		// These three modules form a dependency tree:
		//    ata_generic -> libata -> scsi_mod
		// Try deleting any of the dependencies and see what happens in `make demo`
		//"/lib/modules/5.10.0-19-amd64/kernel/drivers/ata/ata_generic.ko": "",
		//"/lib/modules/5.10.0-19-amd64/kernel/drivers/ata/libata.ko":      "",
		//"/lib/modules/5.10.0-19-amd64/kernel/drivers/scsi/scsi_mod.ko":   "",
	},
	Kmod: []string{
		"e1000",
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

	var wg sync.WaitGroup
	copyQueue := make(chan srcdest)
	wg.Add(1)
	go func() {
		for f := range copyQueue {
			if rc != 0 {
				continue
			}
			fmt.Printf("%s -> %s\n", f.src, f.dest)
			err := initramfs.Copy(f.src, f.dest)
			if err != nil {
				fail(err)
			}
		}
		wg.Done()
	}()

	cp := func(src, dest string) {
		copyQueue <- srcdest{src, dest}
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
	for dest, src := range config.Copy {
		if src == "" {
			src = dest
		}
		if len(dest) > 0 && dest[0] == '/' {
			dest = dest[1:]
		}
		cp(src, dest)
	}
	modules, err := findKernelModules(config.KmodDir, config.Kmod)
	if err != nil {
		return fail(err)
	}
	for _, module := range modules {
		copyQueue <- module
	}

	close(copyQueue)
	wg.Wait()
	return rc
}

type srcdest struct {
	src  string
	dest string
}
