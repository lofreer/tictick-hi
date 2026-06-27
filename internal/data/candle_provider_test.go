package data

import (
	"context"
	"testing"
	"time"
)

func TestCandleProviderReturnsHealthyNativeCandles(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := fakeCandleStore{candles: []Candle{
		testCandle(start, "10", "11", "9", "10.5", "1"),
		testCandle(start.Add(time.Minute), "10.5", "12", "10", "11", "2"),
	}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceNative || result.Health != CandleHealthOK || result.BaseInterval != "1m" {
		t.Fatalf("unexpected metadata: %#v", result)
	}
	if len(result.Candles) != 2 || len(result.Gaps) != 0 {
		t.Fatalf("unexpected candles/gaps: %#v", result)
	}
}

func TestCandleProviderAggregatesFromOneMinuteCandles(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := fakeCandleStore{candles: []Candle{
		testCandle(start, "10", "12", "9", "11", "1"),
		testCandle(start.Add(time.Minute), "11", "13", "10", "12", "2"),
		testCandle(start.Add(2*time.Minute), "12", "12.5", "8", "9", "3"),
		testCandle(start.Add(3*time.Minute), "9", "10", "7", "8", "4"),
		testCandle(start.Add(4*time.Minute), "8", "11", "8", "10", "5"),
	}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || result.Health != CandleHealthOK || result.BaseInterval != "1m" {
		t.Fatalf("unexpected metadata: %#v", result)
	}
	if len(result.Candles) != 1 || result.Candles[0].Interval != "5m" || result.Candles[0].Volume != "15" {
		t.Fatalf("unexpected aggregation: %#v", result.Candles)
	}
}

func TestCandleProviderFallsBackWhenNativeHasGap(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := fakeCandleStore{candles: []Candle{
		testIntervalCandle("5m", start, "10"),
		testIntervalCandle("5m", start.Add(10*time.Minute), "11"),
		testCandle(start, "10", "11", "9", "10", "1"),
		testCandle(start.Add(time.Minute), "10", "11", "9", "10", "1"),
		testCandle(start.Add(2*time.Minute), "10", "11", "9", "10", "1"),
		testCandle(start.Add(3*time.Minute), "10", "11", "9", "10", "1"),
		testCandle(start.Add(4*time.Minute), "10", "11", "9", "10", "1"),
	}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || result.Health != CandleHealthOK {
		t.Fatalf("unexpected fallback result: %#v", result)
	}
}

func TestCandleProviderReportsGaps(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := fakeCandleStore{candles: []Candle{
		testCandle(start, "10", "11", "9", "10", "1"),
		testCandle(start.Add(2*time.Minute), "10", "11", "9", "10", "1"),
	}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Health != CandleHealthGap || len(result.Gaps) != 1 || result.Gaps[0].MissingCandles != 1 {
		t.Fatalf("unexpected gaps: %#v", result)
	}
}

func TestCandleProviderReturnsGappedNativeWhenFallbackIsMissing(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := fakeCandleStore{candles: []Candle{
		testIntervalCandle("5m", start, "10"),
		testIntervalCandle("5m", start.Add(10*time.Minute), "11"),
	}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceNative || result.Health != CandleHealthGap || len(result.Gaps) != 1 {
		t.Fatalf("unexpected native gap result: %#v", result)
	}
}

func TestCandleProviderReportsInsufficientData(t *testing.T) {
	result, err := NewCandleProvider(fakeCandleStore{}).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "15m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceNone || result.Health != CandleHealthInsufficient || len(result.Candles) != 0 {
		t.Fatalf("unexpected insufficient result: %#v", result)
	}
}

type fakeCandleStore struct {
	candles []Candle
}

func (store fakeCandleStore) ListNativeCandles(_ context.Context, query CandleQuery) ([]Candle, error) {
	matches := make([]Candle, 0)
	for _, candle := range store.candles {
		if candle.Exchange == query.Exchange && candle.Symbol == query.Symbol && candle.Interval == query.Interval {
			matches = append(matches, candle)
		}
	}
	return matches, nil
}

func testIntervalCandle(interval string, openTime time.Time, close string) Candle {
	duration, _ := IntervalDuration(interval)
	return Candle{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Interval:  interval,
		OpenTime:  openTime,
		CloseTime: openTime.Add(duration),
		Open:      close,
		High:      close,
		Low:       close,
		Close:     close,
		Volume:    "1",
		IsClosed:  true,
	}
}
