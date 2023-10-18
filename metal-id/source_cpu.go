package metal_id

import (
	"bufio"
	"os"
	"strings"
)

type CpuData struct {
	abstractDataSource
}

func (d *CpuData) Next() []byte {
	if d.IsEmpty() {
		d.fill()
	}
	return d.abstractDataSource.Next()
}

func (d *CpuData) fill() {
	const cpuinfoPath = "/proc/cpuinfo"
	cpuinfo, err := os.Open(cpuinfoPath)
	if err != nil {
		return
	}
	defer func() { _ = cpuinfo.Close() }()
	var cpuCount byte
	scanner := bufio.NewScanner(cpuinfo)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "processor") {
			cpuCount++
		}
		if cpuCount > 1 {
			continue
		}
		for _, field := range []string{
			"cpu family",
			"model name",
			"model",
			"vendor_id",
			"cache size",
		} {
			if strings.HasPrefix(line, field) {
				_, value, found := strings.Cut(line, ":")
				if found {
					d.Append([]byte(strings.TrimSpace(value)))
				}
				break
			}
		}
	}
	d.Append([]byte{'c', 'p', 'u', '0' + cpuCount})
}
