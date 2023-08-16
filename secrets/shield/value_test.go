package shield

import (
	"testing"
)

func TestUnshieldedValue(t *testing.T) {
	const input = "hello world"
	v := UnshieldedValue([]byte(input))
	b := v.Bytes()
	if string(b) != input {
		t.Fatalf("unexpected value: got %q, want %q", string(b), input)
	}
	v.Close()
	if string(b) == input {
		t.Fatalf("unexpected value after close: got %q", string(b))
	}
	if string(v.Bytes()) == input {
		t.Fatalf("unexpected value.Bytes() after close: got %q", string(v.Bytes()))
	}
}
