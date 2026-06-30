package data

import (
	"context"
	"testing"
	"time"
)

func TestCandleProviderReportsAggregationGapAcrossBasePageBoundary(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	targetLimit := MaxCandleLimit/60 + 2
	requiredBaseCandles := targetLimit * 60
	missingBaseIndex := MaxCandleLimit
	candles := make([]Candle, 0, requiredBaseCandles-1)
	for index := 0; index < requiredBaseCandles; index++ {
		if index == missingBaseIndex {
			continue
		}
		openTime := start.Add(time.Duration(index) * time.Minute)
		candles = append(candles, testCandle(openTime, "10", "10", "10", "10", "1"))
	}

	result, err := NewCandleProvider(fakeCandleStore{candles: candles}).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		From:     &start,
		Limit:    targetLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || result.Health != CandleHealthGap {
		t.Fatalf("unexpected metadata: %#v", result)
	}
	if result.Coverage.RequiredBaseCandles != requiredBaseCandles ||
		result.Coverage.BaseLimit != requiredBaseCandles ||
		result.Coverage.ReturnedBaseCandles != requiredBaseCandles-1 ||
		result.Coverage.ReturnedCandles != targetLimit-1 ||
		result.Coverage.LimitedByBaseWindow {
		t.Fatalf("unexpected coverage: %#v", result.Coverage)
	}
	if len(result.Gaps) != 1 {
		t.Fatalf("expected one base gap across page boundary: %#v", result.Gaps)
	}
	missingOpen := start.Add(time.Duration(missingBaseIndex) * time.Minute)
	assertGap(t, result.Gaps[0], missingOpen, missingOpen.Add(time.Minute), 1)

	gappedWindowOpen := alignTime(missingOpen, time.Hour)
	for _, candle := range result.Candles {
		if candle.OpenTime.Equal(gappedWindowOpen) {
			t.Fatalf("aggregated candle for gapped window was returned: %#v", candle)
		}
	}
}
