package data

import (
	"strings"
	"testing"
	"time"
)

func TestValidateCandleSeriesRejectsInvalidCandles(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		candles []Candle
		want    string
	}{
		{
			name: "valid",
			candles: []Candle{
				testCandle(start, "10", "10", "10", "10", "1"),
				testCandle(start.Add(time.Minute), "11", "11", "11", "11", "1"),
			},
		},
		{
			name: "misaligned open time",
			candles: []Candle{
				testCandle(start.Add(30*time.Second), "10", "10", "10", "10", "1"),
			},
			want: "not aligned",
		},
		{
			name: "close time mismatch",
			candles: []Candle{
				func() Candle {
					candle := testCandle(start, "10", "10", "10", "10", "1")
					candle.CloseTime = start.Add(2 * time.Minute)
					return candle
				}(),
			},
			want: "does not match",
		},
		{
			name: "out of order",
			candles: []Candle{
				testCandle(start.Add(time.Minute), "11", "11", "11", "11", "1"),
				testCandle(start, "10", "10", "10", "10", "1"),
			},
			want: "out of order",
		},
		{
			name: "duplicate",
			candles: []Candle{
				testCandle(start, "10", "10", "10", "10", "1"),
				testCandle(start, "11", "11", "11", "11", "1"),
			},
			want: "duplicate",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCandleSeries(tc.candles, "1m")
			if tc.want == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.want != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tc.want)
				}
			}
		})
	}
}
