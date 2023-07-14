package metal_id

import (
	"bytes"
	"os"
	"path/filepath"
)

type UsbDeviceData struct {
	abstractDataSource
}

func (d *UsbDeviceData) Next() []byte {
	if d.IsEmpty() {
		d.fill()
	}
	return d.abstractDataSource.Next()
}

func (d *UsbDeviceData) fill() {
	const sysfs = "/sys/bus/usb/devices"

	subdirs, _ := os.ReadDir(sysfs)
	for _, dir := range subdirs {
		path := filepath.Join(sysfs, dir.Name())
		if !isDir(path) {
			continue
		}
		usb, err := readUSB(path)
		if err != nil {
			continue
		}
		d.Append(usb)
	}
}

func readUSB(path string) ([]byte, error) {
	var endpoints = []string{
		"idVendor",
		"idProduct",
		"version",
		"serial",
		"product",
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
	return bytes.Join(device, nil), nil
}
