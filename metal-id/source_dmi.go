package metal_id

import (
	"os"
	"path/filepath"
)

type DMIData struct {
	abstractDataSource
}

func (d *DMIData) Next() []byte {
	if d.IsEmpty() {
		d.fill()
	}
	return d.abstractDataSource.Next()
}

func (d *DMIData) fill() {
	const sysfs = "/sys/class/dmi/id"
	var endpoints = []string{
		"bios_date",
		"bios_release",
		"bios_version",
		"board_serial",
		"board_vendor",
		"board_version",
		"chassis_vendor",
		"chassis_version",
		"product_name",
		"product_serial",
		"product_version",
		"sys_vendor",
	}

	for _, endpoint := range endpoints {
		content, err := os.ReadFile(filepath.Join(sysfs, endpoint))
		if err != nil {
			continue
		}
		if len(content) == 1 && content[0] == byte('\n') {
			continue
		}
		d.Append(content)
	}
}
