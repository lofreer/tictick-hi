package postgres

import (
	"context"
	"os"
	"strconv"
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
