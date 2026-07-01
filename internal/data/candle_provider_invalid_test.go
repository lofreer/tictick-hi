package data

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCandleProviderReportsInvalidNativeCandleSeries(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := fakeCandleStore{candles: []Candle{
		testCandle(start, "10", "10", "10", "10", "1"),
		testCandle(start, "11", "11", "11", "11", "1"),
	}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceNative || result.Health != CandleHealthInvalid || len(result.Candles) != 0 {
		t.Fatalf("unexpected invalid result: %#v", result)
	}
	if len(result.Issues) != 1 || result.Issues[0].Code != "invalid_native_series" ||
		result.Issues[0].OpenTime == nil ||
		!result.Issues[0].OpenTime.Equal(start) ||
		!strings.Contains(result.Issues[0].Message, "duplicate") {
		t.Fatalf("unexpected issues: %#v", result.Issues)
	}
}

func TestCandleProviderReportsInvalidNativeOutOfOrderOpenTime(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := rawCandleStore{candles: []Candle{
		testCandle(start.Add(time.Minute), "11", "11", "11", "11", "1"),
		testCandle(start, "10", "10", "10", "10", "1"),
	}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceNative || result.Health != CandleHealthInvalid {
		t.Fatalf("unexpected invalid result: %#v", result)
	}
	if len(result.Issues) != 1 ||
		result.Issues[0].Code != "invalid_native_series" ||
		result.Issues[0].OpenTime == nil ||
		!result.Issues[0].OpenTime.Equal(start) ||
		!strings.Contains(result.Issues[0].Message, "out of order") {
		t.Fatalf("unexpected issue: %#v", result.Issues)
	}
}

func TestCandleProviderReportsInvalidNativeCandleValue(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := fakeCandleStore{candles: []Candle{
		testCandle(start, "0", "1", "0", "0.5", "0"),
	}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Health != CandleHealthInvalid || len(result.Candles) != 0 {
		t.Fatalf("unexpected invalid value result: %#v", result)
	}
	if len(result.Issues) != 1 ||
		result.Issues[0].OpenTime == nil ||
		!result.Issues[0].OpenTime.Equal(start) ||
		!strings.Contains(result.Issues[0].Message, "price value must be positive") {
		t.Fatalf("unexpected issue: %#v", result.Issues)
	}
}

func TestCandleProviderReportsInvalidAggregationBaseCandleSeries(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	badBase := testCandle(start, "10", "10", "10", "10", "1")
	badBase.CloseTime = start.Add(2 * time.Minute)
	store := fakeCandleStore{candles: []Candle{badBase}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || result.BaseInterval != "1m" || result.Health != CandleHealthInvalid {
		t.Fatalf("unexpected invalid base result: %#v", result)
	}
	if len(result.Issues) != 1 || result.Issues[0].Code != CandleIssueInvalidCloseTime ||
		result.Issues[0].OpenTime == nil ||
		!result.Issues[0].OpenTime.Equal(start) ||
		!strings.Contains(result.Issues[0].Message, "does not match") {
		t.Fatalf("unexpected base issues: %#v", result.Issues)
	}
}

func TestCandleProviderReportsInvalidAggregationBaseCrossPageOpenTime(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	page := make([]Candle, 0, MaxCandleLimit)
	for index := 0; index < MaxCandleLimit; index++ {
		page = append(page, testCandle(start.Add(time.Duration(index)*time.Minute), "10", "10", "10", "10", "1"))
	}
	store := replayingBasePageStore{page: page}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1d",
		From:     &start,
		Limit:    4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || result.BaseInterval != "1m" || result.Health != CandleHealthInvalid {
		t.Fatalf("unexpected invalid base result: %#v", result)
	}
	if len(result.Issues) != 1 ||
		result.Issues[0].Code != "invalid_aggregation_base_series" ||
		result.Issues[0].OpenTime == nil ||
		!result.Issues[0].OpenTime.Equal(start) ||
		!strings.Contains(result.Issues[0].Message, "out of order") {
		t.Fatalf("unexpected base issue: %#v", result.Issues)
	}
}

type rawCandleStore struct {
	candles []Candle
}

func (store rawCandleStore) ListNativeCandles(_ context.Context, query CandleQuery) ([]Candle, error) {
	return store.matchingCandles(query), nil
}

func (store rawCandleStore) ListLatestNativeCandles(_ context.Context, query CandleQuery) ([]Candle, error) {
	return store.matchingCandles(query), nil
}

func (store rawCandleStore) matchingCandles(query CandleQuery) []Candle {
	matches := make([]Candle, 0)
	for _, candle := range store.candles {
		if candle.Exchange != query.Exchange || candle.Symbol != query.Symbol || candle.Interval != query.Interval {
			continue
		}
		if query.From != nil && candle.OpenTime.Before(*query.From) {
			continue
		}
		if query.To != nil && candle.OpenTime.After(*query.To) {
			continue
		}
		matches = append(matches, candle)
	}
	if limit := NormalizeCandleLimit(query.Limit); len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

type replayingBasePageStore struct {
	page []Candle
}

func (store replayingBasePageStore) ListNativeCandles(_ context.Context, query CandleQuery) ([]Candle, error) {
	if query.Exchange != "binance" || query.Symbol != "BTCUSDT" || query.Interval != baseCandleInterval {
		return nil, nil
	}
	limit := NormalizeCandleLimit(query.Limit)
	if len(store.page) <= limit {
		return append([]Candle(nil), store.page...), nil
	}
	return append([]Candle(nil), store.page[:limit]...), nil
}

func (store replayingBasePageStore) ListLatestNativeCandles(_ context.Context, query CandleQuery) ([]Candle, error) {
	return store.ListNativeCandles(context.Background(), query)
}
