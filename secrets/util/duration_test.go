package util

import (
	"testing"

	"time"
)

func TestParseDuration(t *testing.T) {
	var tests = []struct {
		input    string
		duration time.Duration
	}{
		{"1h2m3s", time.Hour + 2*time.Minute + 3*time.Second},
		{"2m", 2 * time.Minute},
		{"1h", time.Hour},
		{"-1h", -time.Hour},
		{"-1d", -time.Hour * 24},
		{"-1d2h", -(time.Hour * 26)},
		{"2m3s", 2*time.Minute + 3*time.Second},
		{"3s", 3 * time.Second},
		{"2d1h2m3s", time.Hour + 2*time.Minute + 3*time.Second + 2*24*time.Hour},
		{"2d", 2 * 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			duration, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tt.input, err)
			}
			if duration != tt.duration {
				t.Fatalf("incorrectly parsed %s as %v", tt.input, duration)
			}
		})
	}
	var failures = []string{
		"",
		"2d1d",
		"3y",
	}
	for _, input := range failures {
		t.Run(input, func(t *testing.T) {
			duration, err := ParseDuration(input)
			if err == nil {
				t.Fatalf("parsed invalid input %q as %v", input, duration)
			}
		})
	}
}
