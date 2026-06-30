package data

import (
	"context"
	"testing"
	"time"
)

func TestCandleProviderReportsLimitedAggregationCoverage(t *testing.T) {
	originalPages := maxAggregationBasePages
	maxAggregationBasePages = 2
	t.Cleanup(func() { maxAggregationBasePages = originalPages })

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	baseLimit := MaxCandleLimit * maxAggregationBasePages
	store := generatedCandleStore{start: start, count: baseLimit + 1000}

	result, err := NewCandleProvider(store).GetCandles(context.Background(), CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "4h",
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
	if result.Coverage.RequiredBaseCandles != DefaultCandleLimit*240 ||
		result.Coverage.BaseLimit != baseLimit {
		t.Fatalf("unexpected base coverage: %#v", result.Coverage)
	}
	if result.Coverage.ReturnedBaseCandles != baseLimit ||
		result.Coverage.ReturnedCandles >= result.Coverage.RequestedLimit {
		t.Fatalf("unexpected returned coverage: %#v", result.Coverage)
	}
	if !result.Pagination.HasPrevious || result.Pagination.HasNext {
		t.Fatalf("limited aggregation must expose only previous pagination: %#v", result.Pagination)
	}
	if result.Pagination.PreviousCursor == "" || result.Pagination.NextCursor != "" {
		t.Fatalf("limited aggregation must expose an opaque previous cursor only: %#v", result.Pagination)
	}

	previousCursor, err := DecodeCandleCursor(result.Pagination.PreviousCursor)
	if err != nil {
		t.Fatal(err)
	}
	if !previousCursor.MatchesQuery(dataQuery("4h")) || previousCursor.Limit != DefaultCandleLimit {
		t.Fatalf("unexpected limited aggregation previous cursor context: %#v", previousCursor)
	}
}
