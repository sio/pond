package metal_id

import (
	"bytes"
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
		if err != nil {
			return nil, err
		}
		device[index] = content
	}
	return bytes.Join(device, nil), nil
}
