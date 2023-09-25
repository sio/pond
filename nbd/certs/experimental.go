package certs

import (
	"os"
)

func experimental(t skipable) {
	if os.Getenv("RUN_EXPLORATORY_TESTS_WHICH_MAY_LEAVE_GARBAGE_BEHIND") == "" {
		t.Skip("experimental tests are disabled by default")
	}
}

type skipable interface {
	Skip(args ...any)
}
