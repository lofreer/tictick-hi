package data

import (
	"testing"
	"time"
)

func TestClosedCandlesFiltersOpenCandles(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		testCandle(start, "10", "10", "10", "10", "1"),
		testCandle(start.Add(time.Minute), "11", "11", "11", "11", "1"),
		testCandle(start.Add(2*time.Minute), "12", "12", "12", "12", "1"),
	}
	candles[1].IsClosed = false

	closed := ClosedCandles(candles)

	if len(closed) != 2 {
		t.Fatalf("closed len = %d, want 2", len(closed))
	}
	if closed[0].OpenTime != candles[0].OpenTime || closed[1].OpenTime != candles[2].OpenTime {
		t.Fatalf("unexpected closed candles: %#v", closed)
	}
}
