package metal_id

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

type annotatedDataSource struct {
	Name string
	Data DataSource
}

func (ds *annotatedDataSource) Next() []byte {
	return ds.Data.Next()
}

// Order of data sources matters. Increment idVersion after any changes
func Sources() []*annotatedDataSource {
	return []*annotatedDataSource{
		{"Network interfaces", &NetworkInterfacesData{}},
	}
}
