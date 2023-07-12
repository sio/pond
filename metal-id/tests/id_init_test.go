package tests

import (
	"testing"

	"metal_id"

	"fmt"
	"strings"
)

func TestInitialization(t *testing.T) {
	id := metal_id.New()
	_, err := id.Key()
	if err == nil {
		t.Fatalf("Empty MetalID must not be able to derive keys")
	}

	want := "not enough data points"
	got := fmt.Sprint(err)
	if !strings.HasPrefix(got, want) {
		t.Fatalf(`Unexpected error message: want "%s", got "%s"`, want, got)
	}
}
