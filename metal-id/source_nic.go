package metal_id

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
)

const sysfs = "/sys/class/net"

var endpoints = []string{
	"device/vendor",
	"device/device",
	"address",
}

var errNotPhysicalNIC = errors.New("not a physical network controller")

func readNIC(ifname string) ([]byte, error) {
	nic := make([][]byte, len(endpoints))
	for index, endpoint := range endpoints {
		content, err := os.ReadFile(filepath.Join(sysfs, ifname, endpoint))
		if err != nil {
			if os.IsNotExist(err) {
				return nil, errNotPhysicalNIC
			}
			return nil, err
		}
		nic[index] = content
	}
	return bytes.Join(nic, []byte("\t")), nil
}

type NetworkInterfacesData struct {
	abstractDataSource
}

func (d *NetworkInterfacesData) Next() []byte {
	if d.IsEmpty() {
		d.Fill()
	}
	return d.abstractDataSource.Next()
}

func (d *NetworkInterfacesData) Fill() {
	subdirs, _ := os.ReadDir(sysfs)
	for _, dir := range subdirs {
		if !dir.IsDir() {
			continue
		}
		nic, err := readNIC(dir.Name())
		if err != nil && !errors.Is(err, errNotPhysicalNIC) {
			log.Printf("failed to gather data from %s: %v", dir.Name(), err)
			continue
		}
		d.Append(nic)
	}
}
