package metal_id

import (
	"bytes"
	"os"
	"path/filepath"
)

type BlockDeviceData struct {
	abstractDataSource
}

func (d *BlockDeviceData) Next() []byte {
	if d.IsEmpty() {
		d.fill()
	}
	return d.abstractDataSource.Next()
}

func (d *BlockDeviceData) fill() {
	const sysfs = "/sys/block"
	subdirs, _ := os.ReadDir(sysfs)
	for _, dir := range subdirs {
		path := filepath.Join(sysfs, dir.Name())
		if !isDir(path) {
			continue
		}
		device, err := aboutBlockDevice(path)
		if err != nil {
			continue
		}
		d.Append(device)
		disk, err := os.Open(filepath.Join("/dev", dir.Name()))
		if err != nil {
			continue
		}
		lba0 := make([]byte, 512)
		n, err := disk.Read(lba0)
		if err != nil {
			continue
		}
		lba0 = lba0[:n]
		d.Append(lba0)
	}
}

func aboutBlockDevice(path string) ([]byte, error) {
	var endpoints = []string{
		"device/vendor",
		"device/device",
		"size",
		"serial",
	}
	device := make([][]byte, len(endpoints))
	for index, endpoint := range endpoints {
		content, err := os.ReadFile(filepath.Join(path, endpoint))
		if os.IsNotExist(err) {
			return nil, errNotPhysicalDevice
		}
		if err != nil {
			return nil, err
		}
		device[index] = content
	}
	return bytes.Join(device, nil), nil
}
