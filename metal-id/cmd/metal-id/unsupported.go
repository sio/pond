package main

import "runtime"

// All data sources are very Linux-specific
// and rely on information provided by Linux sysfs
func checkOS() {
	if runtime.GOOS == "linux" {
		return
	}
	fail("Unsupported OS: %s", runtime.GOOS)
}
