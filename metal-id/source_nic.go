package metal_id

import (
	"bytes"
	"errors"
	"fmt"
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
		// The following if statement is ignoring best practices on purpose because:
		// - dir.IsDir() does not consider symlinks to directories to be directories
		// - filepath.Join() loses typically meaningless but important for us "/." at the end of path
		if _, err := os.Stat(fmt.Sprintf("%s/%s/.", sysfs, dir.Name())); os.IsNotExist(err) {
			continue
		}
		nic, err := readNIC(dir.Name())
		if errors.Is(err, errNotPhysicalNIC) {
			continue
		}
		if err != nil {
			log.Printf("failed to gather data from %s: %v", dir.Name(), err)
			continue
		}
		d.Append(nic)
	}
}
