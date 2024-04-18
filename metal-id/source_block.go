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
		_ = disk.Close()
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
		"device/model",
		"size",
		"serial",
	}
	_, err := os.Stat(filepath.Join(path, "device"))
	if os.IsNotExist(err) {
		return nil, errNotPhysicalDevice
	}
	device := make([][]byte, 0, len(endpoints))
	for _, endpoint := range endpoints {
		content, err := os.ReadFile(filepath.Join(path, endpoint))
		if err != nil {
			continue
		}
		if len(content) <= 2 { // meaningless '\n' and '0\n'
			continue
		}
		device = append(device, content)
	}
	return bytes.Join(device, nil), nil
}
