package metal_id

import (
	"testing"

	"bytes"
	"crypto"
)

func TestKeyDerivation(t *testing.T) {
	var (
		id  *MetalID
		key crypto.Signer
		err error
	)
	id = New()
	for id.count < dataPointMinCount {
		_, err = id.Key()
		if err == nil {
			t.Fatalf("produced a key from %d data points instead of %d", id.count, dataPointMinCount)
		}
		id.Write(bytes.Repeat([]byte{0x10}, dataPointMinBytes))
	}
	key, err = id.Key()
	if err != nil {
		t.Fatalf("failed to produce a key from %d data points: %v", id.count, err)
	}
	if key == nil {
		t.Fatalf("return nil key after %d data points", id.count)
	}
}
