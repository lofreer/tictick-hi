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
		{
			name: "non decimal value",
			candles: []Candle{
				testCandle(start, "10", "10", "10", "repair", "1"),
			},
			want: "not a decimal",
		},
		{
			name: "negative value",
			candles: []Candle{
				testCandle(start, "10", "10", "10", "10", "-1"),
			},
			want: "negative",
		},
		{
			name: "zero price",
			candles: []Candle{
				testCandle(start, "0", "1", "0.5", "0.5", "0"),
			},
			want: "price value must be positive",
		},
		{
			name: "high below bounds",
			candles: []Candle{
				testCandle(start, "10", "9", "8", "10", "1"),
			},
			want: "high value is below",
		},
		{
			name: "low above bounds",
			candles: []Candle{
				testCandle(start, "10", "12", "11", "10", "1"),
			},
			want: "low value is above",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCandleSeries(tc.candles, "1m")
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

func TestValidateCandleSeriesForTargetRejectsMismatchedTarget(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candle := testCandle(start, "10", "10", "10", "10", "1")
	candle.Symbol = "ETHUSDT"

	err := ValidateCandleSeriesForTarget([]Candle{candle}, "binance", "BTCUSDT", "1m")
	if err == nil {
		t.Fatal("expected target mismatch error")
	}
	if !strings.Contains(err.Error(), "target does not match") {
		t.Fatalf("error = %q, want target mismatch", err.Error())
	}
}
