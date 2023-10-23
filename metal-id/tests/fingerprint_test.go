package tests

import (
	"github.com/sio/pond/metal_id"
	"testing"

	"crypto"
	"fmt"
)

// Ensure that fingerprint is consistent between runs
func TestFingerprint(t *testing.T) {
	tests := []struct {
		name    string
		src     func() map[string]metal_id.DataSource
		skipErr bool
	}{
		{"regular", metal_id.Sources, false},
		{"paranoid", metal_id.SourcesParanoid, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const repeat = 3
			var results [repeat]crypto.Signer
			var first string
			handle := func(err error, msg string) {
				if err == nil {
					return
				}
				if tt.skipErr {
					t.Skipf(msg, err)
				}
				t.Fatalf(msg, err)
			}
			for i := 0; i < len(results); i++ {
				var err error
				hwid := metal_id.New(nil, sprint) // debug: New(t.Logf, sprint)
				err = hwid.Fetch(tt.src())
				handle(err, "Fetching datasource: %v")

				results[i], err = hwid.Key()
				handle(err, "Deriving key from fingerprint: %v")

				encoded, err := metal_id.EncodePublicKey(results[i].Public())
				handle(err, "Encoding public key: %v")

				if i == 0 {
					first = string(encoded)
				}
				if string(encoded) != first {
					t.Errorf("flaky fingerprint: was %s, became %s", first, encoded)
					continue
				}
				t.Log(string(encoded))
			}
		})
	}
}

func sprint(d []byte) string {
	return fmt.Sprintf("% x", d)
}
