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
		"bios_vendor",
		"bios_version",
		"board_asset_tag",
		"board_name",
		"board_serial",
		"board_vendor",
		"board_version",
		"chassis_asset_tag",
		"chassis_serial",
		"chassis_vendor",
		"chassis_version",
		"product_family",
		"product_name",
		"product_serial",
		"product_sku",
		"product_uuid",
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
