package certs

import (
	"os"
)

func experimentsEnabled() bool {
	return os.Getenv("RUN_EXPLORATORY_TESTS_WHICH_MAY_LEAVE_GARBAGE_BEHIND") != ""
}

func experimental(t skipable) {
	if !experimentsEnabled() {
		t.Skip("experimental tests are disabled by default")
	}
}

type skipable interface {
	Skip(args ...any)
}
