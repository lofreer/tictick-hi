package data

import (
	"context"
	"sort"
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
	if result.Coverage.RequestedLimit != DefaultCandleLimit || result.Coverage.ReturnedCandles != 2 {
		t.Fatalf("unexpected coverage: %#v", result.Coverage)
	}
	if result.Window.Count != 2 {
		t.Fatalf("unexpected window count: %#v", result.Window)
	}
	assertTimePtr(t, result.Window.From, start)
	assertTimePtr(t, result.Window.To, start.Add(time.Minute))
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

func TestCandleProviderReportsLimitedAggregationCoverage(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]Candle, 0, MaxCandleLimit+1000)
	for index := 0; index < MaxCandleLimit+1000; index++ {
		candles = append(candles, testCandle(start.Add(time.Duration(index)*time.Minute), "10", "10", "10", "10", "1"))
	}
	store := fakeCandleStore{candles: candles}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		Limit:    DefaultCandleLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || result.Health != CandleHealthInsufficient {
		t.Fatalf("unexpected metadata: %#v", result)
	}
	if !result.Coverage.LimitedByBaseWindow {
		t.Fatalf("expected limited coverage: %#v", result.Coverage)
	}
	if result.Coverage.RequiredBaseCandles != DefaultCandleLimit*60 || result.Coverage.BaseLimit != MaxCandleLimit {
		t.Fatalf("unexpected base coverage: %#v", result.Coverage)
	}
	if result.Coverage.ReturnedBaseCandles != MaxCandleLimit || result.Coverage.ReturnedCandles >= result.Coverage.RequestedLimit {
		t.Fatalf("unexpected returned coverage: %#v", result.Coverage)
	}
}

func TestCandleProviderReportsNativePagination(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]Candle, 0, 5)
	for index := 0; index < 5; index++ {
		candles = append(candles, testCandle(start.Add(time.Duration(index)*time.Minute), "10", "10", "10", "10", "1"))
	}
	store := fakeCandleStore{candles: candles}
	from := start.Add(time.Minute)

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     &from,
		Limit:    2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candles) != 2 {
		t.Fatalf("unexpected candles: %#v", result.Candles)
	}
	if !result.Pagination.HasPrevious || !result.Pagination.HasNext {
		t.Fatalf("expected previous and next windows: %#v", result.Pagination)
	}
	if result.Pagination.PreviousCursor == "" || result.Pagination.NextCursor == "" {
		t.Fatalf("expected opaque cursors: %#v", result.Pagination)
	}
	assertTimePtr(t, result.Pagination.PreviousFrom, start.Add(-time.Minute))
	assertTimePtr(t, result.Pagination.PreviousTo, start)
	assertTimePtr(t, result.Pagination.NextFrom, start.Add(3*time.Minute))
	assertTimePtr(t, result.Pagination.NextTo, start.Add(4*time.Minute))

	nextCursor, err := DecodeCandleCursor(result.Pagination.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if !nextCursor.MatchesQuery(dataQuery("1m")) || nextCursor.Limit != 2 {
		t.Fatalf("unexpected next cursor context: %#v", nextCursor)
	}
}

func TestCandleProviderReportsAggregatedPaginationFromBaseCandles(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]Candle, 0, 15)
	for index := 0; index < 15; index++ {
		candles = append(candles, testCandle(start.Add(time.Duration(index)*time.Minute), "10", "10", "10", "10", "1"))
	}
	store := fakeCandleStore{candles: candles}
	from := start.Add(5 * time.Minute)

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "5m",
		From:     &from,
		Limit:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || len(result.Candles) != 1 {
		t.Fatalf("unexpected aggregation: %#v", result)
	}
	if result.Window.Count != 1 {
		t.Fatalf("unexpected window count: %#v", result.Window)
	}
	assertTimePtr(t, result.Window.From, start.Add(5*time.Minute))
	assertTimePtr(t, result.Window.To, start.Add(5*time.Minute))
	if !result.Pagination.HasPrevious || !result.Pagination.HasNext {
		t.Fatalf("expected previous and next windows: %#v", result.Pagination)
	}
	if result.Pagination.PreviousCursor == "" || result.Pagination.NextCursor == "" {
		t.Fatalf("expected opaque cursors: %#v", result.Pagination)
	}
	assertTimePtr(t, result.Pagination.PreviousFrom, start)
	assertTimePtr(t, result.Pagination.PreviousTo, start)
	assertTimePtr(t, result.Pagination.NextFrom, start.Add(10*time.Minute))
	assertTimePtr(t, result.Pagination.NextTo, start.Add(10*time.Minute))
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

func TestCandleProviderReportsRangeBoundaryGaps(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := start.Add(4 * time.Minute)
	store := fakeCandleStore{candles: []Candle{
		testCandle(start.Add(time.Minute), "10", "11", "9", "10", "1"),
		testCandle(start.Add(2*time.Minute), "10", "11", "9", "10", "1"),
		testCandle(start.Add(3*time.Minute), "10", "11", "9", "10", "1"),
	}}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     &start,
		To:       &to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Health != CandleHealthGap || len(result.Gaps) != 2 {
		t.Fatalf("unexpected boundary gaps: %#v", result)
	}
	assertGap(t, result.Gaps[0], start, start.Add(time.Minute), 1)
	assertGap(t, result.Gaps[1], start.Add(4*time.Minute), start.Add(5*time.Minute), 1)
}

func TestCandleProviderReportsWholeRequestedWindowGap(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := start.Add(2 * time.Minute)

	result, err := NewCandleProvider(fakeCandleStore{}).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     &start,
		To:       &to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceNone || result.Health != CandleHealthGap || len(result.Gaps) != 1 {
		t.Fatalf("unexpected empty range result: %#v", result)
	}
	assertGap(t, result.Gaps[0], start, start.Add(3*time.Minute), 3)
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
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})
	if limit := NormalizeCandleLimit(query.Limit); len(matches) > limit {
		matches = matches[:limit]
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

func assertTimePtr(t *testing.T, actual *time.Time, expected time.Time) {
	t.Helper()
	if actual == nil {
		t.Fatalf("time pointer is nil, want %s", expected.Format(time.RFC3339))
	}
	if !actual.Equal(expected) {
		t.Fatalf("time = %s, want %s", actual.Format(time.RFC3339), expected.Format(time.RFC3339))
	}
}

func assertGap(t *testing.T, actual CandleGap, from time.Time, to time.Time, missingCandles int) {
	t.Helper()
	if !actual.From.Equal(from) || !actual.To.Equal(to) || actual.MissingCandles != missingCandles {
		t.Fatalf("gap = %#v, want from %s to %s missing %d", actual, from, to, missingCandles)
	}
}

func dataQuery(interval string) CandleQuery {
	return CandleQuery{Exchange: "binance", Symbol: "BTCUSDT", Interval: interval}
}
