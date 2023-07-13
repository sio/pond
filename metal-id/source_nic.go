package metal_id

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
)

type NetworkInterfacesData struct {
	abstractDataSource
}

func (d *NetworkInterfacesData) Next() []byte {
	if d.IsEmpty() {
		d.fill()
	}
	return d.abstractDataSource.Next()
}

func (d *NetworkInterfacesData) fill() {
	const sysfs = "/sys/class/net"
	subdirs, _ := os.ReadDir(sysfs)
	for _, dir := range subdirs {
		path := filepath.Join(sysfs, dir.Name())
		if !isDir(path) {
			continue
		}
		nic, err := readNIC(path)
		if errors.Is(err, errNotPhysicalNIC) {
			continue
		}
		if err != nil {
			log.Printf("failed to gather data from %s: %v", path, err)
			continue
		}
		d.Append(nic)
	}
}

func readNIC(path string) ([]byte, error) {
	var endpoints = []string{
		"device/vendor",
		"device/device",
		"address",
	}
	nic := make([][]byte, len(endpoints))
	for index, endpoint := range endpoints {
		content, err := os.ReadFile(filepath.Join(path, endpoint))
		if os.IsNotExist(err) {
			return nil, errNotPhysicalNIC
		}
		if err != nil {
			return nil, err
		}
		nic[index] = content
	}
	return bytes.Join(nic, []byte("\t")), nil
}

var errNotPhysicalNIC = errors.New("not a physical network controller")
