package metal_id

import (
	"bytes"
	"sort"
)

// Data sources used for hardware fingerprinting
//
// Order of sources matters
func Sources() []*annotatedDataSource {
	return []*annotatedDataSource{
		{"Network interfaces", &NetworkInterfacesData{}},
		{"PCI devices", &PciDeviceData{}},
		{"USB devices", &UsbDeviceData{}},
		{"DMI table", &DMIData{}},
	}
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
}

// Adding some human-friendly annotations for interactive use and debugging
type annotatedDataSource struct {
	Name string
	Data DataSource
}

func (ds *annotatedDataSource) Next() []byte {
	return ds.Data.Next()
}

// Abstract data source that implements all the boring routines
//
// Used internally to implement most of default data sources.
// Intentionally not exported because it is too rigid and too opinionated to be
// considered a reference implementation.
type abstractDataSource struct {
	chunks [][]byte
	index  int
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
	if len(d.chunks) == 0 {
		d.index = 0xEE // any number greater than zero will stop iteration
	}
}
