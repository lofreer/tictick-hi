package postgres

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationCandleProviderReportsPaginationWindows(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	symbol := integrationSymbol("PG")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	start := time.Date(2026, 6, 27, 3, 0, 0, 0, time.UTC)
	for index := 0; index < 10; index++ {
		insertIntegrationCandle(t, ctx, store, data.Candle{
			Exchange:  "binance",
			Symbol:    symbol,
			Interval:  "1m",
			OpenTime:  start.Add(time.Duration(index) * time.Minute),
			CloseTime: start.Add(time.Duration(index+1) * time.Minute),
			Open:      fmt.Sprintf("%d", 100+index),
			High:      fmt.Sprintf("%d", 101+index),
			Low:       fmt.Sprintf("%d", 99+index),
			Close:     fmt.Sprintf("%d", 100+index),
			Volume:    "1",
			IsClosed:  true,
		})
	}

	latest, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !latest.Pagination.HasPrevious || latest.Pagination.HasNext {
		t.Fatalf("unexpected latest pagination: %#v", latest.Pagination)
	}
	assertTimePtr(t, latest.Pagination.PreviousFrom, start.Add(4*time.Minute))
	assertTimePtr(t, latest.Pagination.PreviousTo, start.Add(6*time.Minute))

	from := start.Add(2 * time.Minute)
	inRange, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		From:     &from,
		Limit:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !inRange.Pagination.HasPrevious || !inRange.Pagination.HasNext {
		t.Fatalf("unexpected range pagination: %#v", inRange.Pagination)
	}
	assertTimePtr(t, inRange.Pagination.PreviousFrom, start.Add(-time.Minute))
	assertTimePtr(t, inRange.Pagination.PreviousTo, start.Add(time.Minute))
	assertTimePtr(t, inRange.Pagination.NextFrom, start.Add(5*time.Minute))
	assertTimePtr(t, inRange.Pagination.NextTo, start.Add(7*time.Minute))

	aggregated, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "5m",
		Limit:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !aggregated.Pagination.HasPrevious || aggregated.Pagination.HasNext {
		t.Fatalf("unexpected aggregated pagination: %#v", aggregated.Pagination)
	}
}

func TestIntegrationListNativeCandlesUsesLatestWindowBeforeTo(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	symbol := integrationSymbol("LT")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	start := time.Date(2026, 6, 27, 3, 30, 0, 0, time.UTC)
	for index := 0; index < 10; index++ {
		insertIntegrationCandle(t, ctx, store, data.Candle{
			Exchange:  "binance",
			Symbol:    symbol,
			Interval:  "1m",
			OpenTime:  start.Add(time.Duration(index) * time.Minute),
			CloseTime: start.Add(time.Duration(index+1) * time.Minute),
			Open:      fmt.Sprintf("%d", 100+index),
			High:      fmt.Sprintf("%d", 101+index),
			Low:       fmt.Sprintf("%d", 99+index),
			Close:     fmt.Sprintf("%d", 100+index),
			Volume:    "1",
			IsClosed:  true,
		})
	}

	to := start.Add(6 * time.Minute)
	latestBefore, err := store.ListLatestNativeCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		To:       &to,
		Limit:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertOpenTimes(t, latestBefore, start.Add(4*time.Minute), start.Add(5*time.Minute), start.Add(6*time.Minute))
}

func TestIntegrationCandleProviderReportsRequestedRangeBoundaryGaps(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	symbol := integrationSymbol("BG")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	start := time.Date(2026, 6, 27, 4, 0, 0, 0, time.UTC)
	for index := 1; index <= 3; index++ {
		insertIntegrationCandle(t, ctx, store, data.Candle{
			Exchange:  "binance",
			Symbol:    symbol,
			Interval:  "1m",
			OpenTime:  start.Add(time.Duration(index) * time.Minute),
			CloseTime: start.Add(time.Duration(index+1) * time.Minute),
			Open:      fmt.Sprintf("%d", 100+index),
			High:      fmt.Sprintf("%d", 101+index),
			Low:       fmt.Sprintf("%d", 99+index),
			Close:     fmt.Sprintf("%d", 100+index),
			Volume:    "1",
			IsClosed:  true,
		})
	}

	to := start.Add(4 * time.Minute)
	result, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		From:     &start,
		To:       &to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Health != data.CandleHealthGap || len(result.Gaps) != 2 {
		t.Fatalf("unexpected boundary gap result: %#v", result)
	}
	assertTaskGap(t, result.Gaps[0], start, start.Add(time.Minute), 1)
	assertTaskGap(t, result.Gaps[1], start.Add(4*time.Minute), start.Add(5*time.Minute), 1)
}

func TestIntegrationCandleProviderRejectsOversizedRepositoryRange(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 5, 0, 0, 0, time.UTC)
	to := start.Add(data.MaxCandleQuerySpan(time.Minute) + time.Minute)
	_, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   integrationSymbol("OR"),
		Interval: "1m",
		From:     &start,
		To:       &to,
	})
	if err == nil || !strings.Contains(err.Error(), "time range must cover at most") {
		t.Fatalf("err = %v, want oversized range error", err)
	}
}

func TestIntegrationCandleProviderRejectsMissingRepositoryTarget(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	_, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Interval: "1m",
	})
	if err == nil || !strings.Contains(err.Error(), "exchange, symbol and interval are required") {
		t.Fatalf("err = %v, want missing target error", err)
	}
}
