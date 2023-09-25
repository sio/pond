package certs

import (
	"testing"
)

func TestPKI(t *testing.T) {
	experimental(t)

	_, err := PublicKey("testkeys/alice.pub")
	if err != nil {
		t.Fatal(err)
	}
}
