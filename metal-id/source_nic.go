package metal_id

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
)

type NetworkInterfacesData struct {
	nics  [][]byte
	index int
}

var endpoints = []string{
	"device/vendor",
	"device/device",
	"address",
}

const (
	sysfs = "/sys/class/net"
)

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

func (data *NetworkInterfacesData) Next() []byte {
	if data.nics == nil && data.index == 0 {
		data.init()
	}
	if data.index > len(data.nics)-1 {
		return nil
	}
	result := data.nics[data.index]
	data.index++
	return result
}

func (data *NetworkInterfacesData) init() {
	if len(data.nics) != 0 {
		return
	}
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
		data.nics = append(data.nics, nic)
	}
	sort.SliceStable(
		data.nics,
		func(i, j int) bool { return bytes.Compare(data.nics[i], data.nics[j]) < 0 },
	)
	if len(data.nics) == 0 {
		data.index = 0xEE // any number greater than zero
	}
}
