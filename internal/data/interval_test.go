package data

import (
	"testing"
	"time"
)

func TestIntervalDuration(t *testing.T) {
	tests := map[string]time.Duration{
		"1m":  time.Minute,
		"15m": 15 * time.Minute,
		"1h":  time.Hour,
		"4h":  4 * time.Hour,
		"1d":  24 * time.Hour,
	}

	for interval, expected := range tests {
		actual, err := IntervalDuration(interval)
		if err != nil {
			t.Fatalf("IntervalDuration(%q) error: %v", interval, err)
		}
		if actual != expected {
			t.Fatalf("IntervalDuration(%q) = %v, want %v", interval, actual, expected)
		}
	}
}
