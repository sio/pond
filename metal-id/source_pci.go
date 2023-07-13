package metal_id

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type PciDeviceData struct {
	abstractDataSource
}

func (d *PciDeviceData) Next() []byte {
	if d.IsEmpty() {
		d.fill()
	}
	return d.abstractDataSource.Next()
}

func (d *PciDeviceData) fill() {
	const sysfs = "/sys/bus/pci/devices"

	subdirs, _ := os.ReadDir(sysfs)
	for _, dir := range subdirs {
		path := filepath.Join(sysfs, dir.Name())
		if !isDir(path) {
			continue
		}
		pci, err := readPCI(path)
		if err != nil {
			log.Printf("failed to gather data from %s: %v", path, err)
			continue
		}
		d.Append(pci)
	}
}

func readPCI(path string) ([]byte, error) {
	var endpoints = []string{
		"vendor",
		"device",
		"revision",
	}
	device := make([][]byte, len(endpoints))
	for index, endpoint := range endpoints {
		content, err := os.ReadFile(filepath.Join(path, endpoint))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		device[index] = content
	}
	rom := readROM(path)
	if len(rom) > 0 {
		device = append(device, rom)
	}
	return bytes.Join(device, nil), nil
}

// Read PCI ROM if available
// https://mjmwired.net/kernel/Documentation/filesystems/sysfs-pci.txt
func readROM(path string) []byte {
	romPath := filepath.Join(path, "rom")
	var stat os.FileInfo
	var err error
	if stat, err = os.Stat(romPath); os.IsNotExist(err) {
		return nil
	}

	rom, err := os.OpenFile(romPath, os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	defer func() { _ = rom.Close() }()

	_, err = fmt.Fprintln(rom, "1")
	if err != nil {
		return nil
	}
	defer func() {
		// Simply writing into previously opened descriptor
		// does not disable ROM reading, so we close it and reopen
		_ = rom.Close()
		rom, _ = os.OpenFile(romPath, os.O_RDWR, 0)
		_, _ = fmt.Fprintln(rom, "0")
		_ = rom.Close()
	}()

	const chunkSize int = 1024

	// First bytes are boring. Let's skip those
	var romSize = int(stat.Size())
	var skip = 0
	if romSize > 3*chunkSize {
		skip = ((romSize / chunkSize) % chunkSize) + (romSize % chunkSize) + chunkSize
	}
	_, err = rom.Seek(int64(skip), 1)
	if err != nil {
		return nil
	}

	var output = make([]byte, chunkSize) // we don't need full ROM
	n, err := rom.Read(output)
	if err != nil {
		return nil
	}
	return output[:n]
}
