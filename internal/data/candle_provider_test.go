package data

import (
	"context"
	"sort"
	"strings"
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

func TestCandleProviderAggregatesAcrossPagedBaseWindow(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	requiredBaseCandles := DefaultCandleLimit * 60
	store := generatedCandleStore{start: start, count: requiredBaseCandles}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		Limit:    DefaultCandleLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || result.Health != CandleHealthOK {
		t.Fatalf("unexpected metadata: %#v", result)
	}
	if result.Coverage.LimitedByBaseWindow {
		t.Fatalf("expected full paged base coverage: %#v", result.Coverage)
	}
	if result.Coverage.RequiredBaseCandles != requiredBaseCandles ||
		result.Coverage.BaseLimit != requiredBaseCandles ||
		result.Coverage.ReturnedBaseCandles != requiredBaseCandles ||
		result.Coverage.ReturnedCandles != DefaultCandleLimit {
		t.Fatalf("unexpected coverage: %#v", result.Coverage)
	}
	assertTimePtr(t, result.Window.From, start)
	assertTimePtr(t, result.Window.To, start.Add(time.Duration(DefaultCandleLimit-1)*time.Hour))
}

func TestCandleProviderAggregatesDailyWindowAcrossStreamingBasePages(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	requiredBaseCandles := DefaultCandleLimit * 24 * 60
	store := generatedCandleStore{start: start, count: requiredBaseCandles}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1d",
		Limit:    DefaultCandleLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || result.Health != CandleHealthOK {
		t.Fatalf("unexpected metadata: %#v", result)
	}
	if len(result.Candles) != DefaultCandleLimit {
		t.Fatalf("unexpected candle count: %d", len(result.Candles))
	}
	if result.Coverage.RequiredBaseCandles != requiredBaseCandles ||
		result.Coverage.BaseLimit != requiredBaseCandles ||
		result.Coverage.ReturnedBaseCandles != requiredBaseCandles ||
		result.Coverage.ReturnedCandles != DefaultCandleLimit ||
		result.Coverage.LimitedByBaseWindow {
		t.Fatalf("unexpected coverage: %#v", result.Coverage)
	}
	assertTimePtr(t, result.Window.From, start)
	assertTimePtr(t, result.Window.To, start.Add(time.Duration(DefaultCandleLimit-1)*24*time.Hour))
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

func TestCandleProviderLatestAggregationPreservesRequestedToBoundaryGap(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]Candle, 0, 5)
	for index := 0; index < 5; index++ {
		candles = append(candles, testCandle(start.Add(time.Duration(index)*time.Minute), "10", "10", "10", "10", "1"))
	}
	store := fakeCandleStore{candles: candles}
	to := start.Add(9 * time.Minute)

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "5m",
		To:       &to,
		Limit:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != CandleSourceAggregated || result.Health != CandleHealthGap {
		t.Fatalf("unexpected metadata: %#v", result)
	}
	if len(result.Gaps) != 1 {
		t.Fatalf("expected one tail gap: %#v", result.Gaps)
	}
	assertGap(t, result.Gaps[0], start.Add(5*time.Minute), start.Add(10*time.Minute), 5)
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

func TestCandleProviderRejectsInvalidRangeBeforeStoreRead(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tooFar := start.Add(MaxCandleQuerySpan(time.Minute) + time.Minute)
	store := &recordingCandleStore{}

	_, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     &start,
		To:       &tooFar,
	})
	if err == nil || !strings.Contains(err.Error(), "time range must cover at most") {
		t.Fatalf("err = %v, want oversized range error", err)
	}
	if store.nativeCalls != 0 || store.latestCalls != 0 {
		t.Fatalf("provider read store before rejecting invalid range: native=%d latest=%d", store.nativeCalls, store.latestCalls)
	}
}

func TestCandleProviderRejectsMissingTargetBeforeStoreRead(t *testing.T) {
	store := &recordingCandleStore{}

	_, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Interval: "1m",
	})
	if err == nil || !strings.Contains(err.Error(), "exchange, symbol and interval are required") {
		t.Fatalf("err = %v, want missing target error", err)
	}
	if store.nativeCalls != 0 || store.latestCalls != 0 {
		t.Fatalf("provider read store before rejecting missing target: native=%d latest=%d", store.nativeCalls, store.latestCalls)
	}
}

type fakeCandleStore struct {
	candles []Candle
}

func (store fakeCandleStore) ListNativeCandles(_ context.Context, query CandleQuery) ([]Candle, error) {
	if query.From == nil && query.To == nil {
		return store.latestNativeCandles(query), nil
	}
	return store.nativeCandles(query), nil
}

func (store fakeCandleStore) ListLatestNativeCandles(_ context.Context, query CandleQuery) ([]Candle, error) {
	return store.latestNativeCandles(query), nil
}

func (store fakeCandleStore) nativeCandles(query CandleQuery) []Candle {
	matches := store.matchingNativeCandles(query)
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})
	if limit := NormalizeCandleLimit(query.Limit); len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

func (store fakeCandleStore) latestNativeCandles(query CandleQuery) []Candle {
	matches := store.matchingNativeCandles(query)
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.After(matches[right].OpenTime)
	})
	if limit := NormalizeCandleLimit(query.Limit); len(matches) > limit {
		matches = matches[:limit]
	}
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})
	return matches
}

func (store fakeCandleStore) matchingNativeCandles(query CandleQuery) []Candle {
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
	return matches
}

type generatedCandleStore struct {
	start time.Time
	count int
}

func (store generatedCandleStore) ListNativeCandles(_ context.Context, query CandleQuery) ([]Candle, error) {
	if query.From == nil && query.To == nil {
		return store.ListLatestNativeCandles(context.Background(), query)
	}
	if query.Exchange != "binance" || query.Symbol != "BTCUSDT" || query.Interval != baseCandleInterval {
		return nil, nil
	}

	firstIndex := 0
	lastIndex := store.count - 1
	if query.From != nil {
		firstIndex = max(firstIndex, ceilMinuteIndex(store.start, query.From.UTC()))
	}
	if query.To != nil {
		lastIndex = min(lastIndex, floorMinuteIndex(store.start, query.To.UTC()))
	}
	return store.candles(firstIndex, lastIndex, NormalizeCandleLimit(query.Limit)), nil
}

func (store generatedCandleStore) ListLatestNativeCandles(_ context.Context, query CandleQuery) ([]Candle, error) {
	if query.Exchange != "binance" || query.Symbol != "BTCUSDT" || query.Interval != baseCandleInterval {
		return nil, nil
	}

	lastIndex := store.count - 1
	if query.To != nil {
		lastIndex = min(lastIndex, floorMinuteIndex(store.start, query.To.UTC()))
	}
	limit := NormalizeCandleLimit(query.Limit)
	firstIndex := max(0, lastIndex-limit+1)
	return store.candles(firstIndex, lastIndex, limit), nil
}

func (store generatedCandleStore) candles(firstIndex int, lastIndex int, limit int) []Candle {
	if store.count <= 0 || lastIndex < firstIndex || lastIndex < 0 || firstIndex >= store.count {
		return nil
	}
	if firstIndex < 0 {
		firstIndex = 0
	}
	if lastIndex >= store.count {
		lastIndex = store.count - 1
	}
	if limit <= 0 {
		return nil
	}
	total := lastIndex - firstIndex + 1
	if total > limit {
		total = limit
	}
	candles := make([]Candle, 0, total)
	for offset := 0; offset < total; offset++ {
		index := firstIndex + offset
		candles = append(candles, testCandle(store.start.Add(time.Duration(index)*time.Minute), "10", "10", "10", "10", "1"))
	}
	return candles
}

func ceilMinuteIndex(start time.Time, value time.Time) int {
	diff := value.Sub(start)
	if diff <= 0 {
		return 0
	}
	index := int(diff / time.Minute)
	if start.Add(time.Duration(index) * time.Minute).Before(value) {
		index++
	}
	return index
}

func floorMinuteIndex(start time.Time, value time.Time) int {
	diff := value.Sub(start)
	if diff < 0 {
		return -1
	}
	return int(diff / time.Minute)
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

type recordingCandleStore struct {
	nativeCalls int
	latestCalls int
}

func (store *recordingCandleStore) ListNativeCandles(context.Context, CandleQuery) ([]Candle, error) {
	store.nativeCalls++
	return nil, nil
}

func (store *recordingCandleStore) ListLatestNativeCandles(context.Context, CandleQuery) ([]Candle, error) {
	store.latestCalls++
	return nil, nil
}
