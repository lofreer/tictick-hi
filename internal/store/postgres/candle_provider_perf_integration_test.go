package postgres

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationCandleProviderLargeAggregationWindowPerformance(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	symbol := integrationSymbol("CPF")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	start := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	targetInterval := "4h"
	targetDuration, err := data.IntervalDuration(targetInterval)
	if err != nil {
		t.Fatal(err)
	}
	requiredBaseCandles := data.DefaultCandleLimit * int(targetDuration/time.Minute)
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		SELECT 'binance',
		       $1,
		       '1m',
		       $2::timestamptz + (series.idx * interval '1 minute'),
		       $2::timestamptz + ((series.idx + 1) * interval '1 minute'),
		       (10000 + series.idx)::numeric,
		       (10001 + series.idx)::numeric,
		       (9999 + series.idx)::numeric,
		       (10000 + series.idx)::numeric,
		       1,
		       true,
		       now()
		  FROM generate_series(0, $3::int - 1) AS series(idx)`,
		symbol,
		start,
		requiredBaseCandles,
	); err != nil {
		t.Fatal(err)
	}

	threshold := candlePerformanceThreshold(t)
	startedAt := time.Now()
	result, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: targetInterval,
		Limit:    data.DefaultCandleLimit,
	})
	elapsed := time.Since(startedAt)
	if err != nil {
		t.Fatal(err)
	}

	if result.Source != data.CandleSourceAggregated ||
		result.Health != data.CandleHealthOK ||
		result.BaseInterval != "1m" ||
		len(result.Candles) != data.DefaultCandleLimit {
		t.Fatalf("unexpected result metadata: %#v", result)
	}
	if result.Coverage.RequiredBaseCandles != requiredBaseCandles ||
		result.Coverage.BaseLimit != requiredBaseCandles ||
		result.Coverage.ReturnedBaseCandles != requiredBaseCandles ||
		result.Coverage.ReturnedCandles != data.DefaultCandleLimit ||
		result.Coverage.LimitedByBaseWindow {
		t.Fatalf("unexpected coverage: %#v", result.Coverage)
	}
	assertTimePtr(t, result.Window.From, start)
	assertTimePtr(t, result.Window.To, start.Add(time.Duration(data.DefaultCandleLimit-1)*targetDuration))
	if elapsed > threshold {
		t.Fatalf("large aggregation query took %s, threshold %s", elapsed, threshold)
	}
	t.Logf("large aggregation query read %d base candles into %d %s candles in %s", result.Coverage.ReturnedBaseCandles, result.Coverage.ReturnedCandles, result.RequestedInterval, elapsed)
}

func TestIntegrationCandleProviderConcurrentAggregationQueries(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	symbol := integrationSymbol("CPC")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	start := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	baseCandles := 2880
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		SELECT 'binance',
		       $1,
		       '1m',
		       $2::timestamptz + (series.idx * interval '1 minute'),
		       $2::timestamptz + ((series.idx + 1) * interval '1 minute'),
		       (50000 + series.idx)::numeric,
		       (50001 + series.idx)::numeric,
		       (49999 + series.idx)::numeric,
		       (50000 + series.idx)::numeric,
		       1,
		       true,
		       now()
		  FROM generate_series(0, $3::int - 1) AS series(idx)`,
		symbol,
		start,
		baseCandles,
	); err != nil {
		t.Fatal(err)
	}

	const workers = 6
	results := make(chan data.CandleResult, workers)
	errors := make(chan error, workers)
	var waitGroup sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			result, err := store.GetCandles(ctx, data.CandleQuery{
				Exchange: "binance",
				Symbol:   symbol,
				Interval: "15m",
				Limit:    96,
			})
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errors)

	for err := range errors {
		t.Fatal(err)
	}
	if len(results) != workers {
		t.Fatalf("concurrent query result count = %d, want %d", len(results), workers)
	}

	expectedFrom := start.Add(24 * time.Hour)
	expectedTo := start.Add(47*time.Hour + 45*time.Minute)
	expectedRequiredBaseCandles := 96 * 15
	for result := range results {
		if result.Source != data.CandleSourceAggregated ||
			result.Health != data.CandleHealthOK ||
			result.BaseInterval != "1m" ||
			len(result.Candles) != 96 {
			t.Fatalf("unexpected concurrent aggregation result: %#v", result)
		}
		if result.Coverage.RequiredBaseCandles != expectedRequiredBaseCandles ||
			result.Coverage.BaseLimit != expectedRequiredBaseCandles ||
			result.Coverage.ReturnedBaseCandles != expectedRequiredBaseCandles ||
			result.Coverage.ReturnedCandles != 96 ||
			result.Coverage.LimitedByBaseWindow {
			t.Fatalf("unexpected concurrent aggregation coverage: %#v", result.Coverage)
		}
		assertTimePtr(t, result.Window.From, expectedFrom)
		assertTimePtr(t, result.Window.To, expectedTo)
		if !result.Pagination.HasPrevious || result.Pagination.HasNext {
			t.Fatalf("unexpected concurrent aggregation pagination: %#v", result.Pagination)
		}
	}
}

func candlePerformanceThreshold(t *testing.T) time.Duration {
	t.Helper()

	raw := os.Getenv("TICTICK_CANDLE_PERF_MAX_MS")
	if raw == "" {
		return 10 * time.Second
	}
	milliseconds, err := strconv.Atoi(raw)
	if err != nil || milliseconds <= 0 {
		t.Fatalf("TICTICK_CANDLE_PERF_MAX_MS must be a positive integer, got %q", raw)
	}
	return time.Duration(milliseconds) * time.Millisecond
}
