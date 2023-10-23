package metal_id

import (
	"bytes"
	"sort"
)

// Data sources used for hardware fingerprinting
//
// # Order of sources matters
//
// These data sources should fail gracefully and should return non-nil .Err()
// only in exceptional situations.
func Sources() map[string]DataSource {
	return map[string]DataSource{
		"Network interfaces": &NetworkInterfacesData{},
		"PCI devices":        &PciDeviceData{},
		"USB devices":        &UsbDeviceData{},
		"DMI table":          &DMIData{},
		"Block devices":      &BlockDeviceData{},
		"CPU information":    &CpuData{},
	}
}

// Extra data sources for paranoid users
//
// These data sources may fail loudly via returning non-nil .Err() at their
// discretion
func SourcesParanoid() map[string]DataSource {
	src := Sources()
	src["Default gateway"] = new(GatewayData)
	return src
}

// Data source provides hardware based inputs for MetalID to uniquely
// fingerprint current machine
type DataSource interface {
	// Multiple data points will be consumed by subsequent Next() calls.
	// Data points must be sorted using a stable sorting algorithm.
	//
	// After the last data point has been consumed all subsequent calls to
	// Next() must return nil
	Next() []byte

	// Datasource may return an error via this method.
	// Callers should check if error occured after receiving nil value via Next()
	Err() error
}

// Abstract data source that implements all the boring routines
//
// Used internally to implement most of default data sources.
// Intentionally not exported because it is too rigid and too opinionated to be
// considered a reference implementation.
type abstractDataSource struct {
	chunks [][]byte
	index  int
	err    error
}

func (d *abstractDataSource) IsEmpty() bool {
	return len(d.chunks) == 0
}

func (d *abstractDataSource) Append(c []byte) {
	if len(c) == 0 {
		return
	}
	d.chunks = append(d.chunks, c)
}

func (d *abstractDataSource) Reset() {
	d.chunks = nil
	d.index = 0
}

func (d *abstractDataSource) Next() []byte {
	if d.index == 0 && len(d.chunks) > 0 {
		d.sort()
	}
	if d.index > len(d.chunks)-1 {
		return nil
	}
	result := d.chunks[d.index]
	d.index++
	return result
}

func (d *abstractDataSource) sort() {
	if len(d.chunks) == 0 {
		return
	}
	sort.SliceStable(
		d.chunks,
		func(i, j int) bool { return bytes.Compare(d.chunks[i], d.chunks[j]) < 0 },
	)
}

func (d *abstractDataSource) Err() error {
	return d.err
}
