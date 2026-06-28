package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationScanMarketCandleGapsReportsPersistedHistory(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 6, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCG")
	cleanupMarketCandles(t, store, symbol)
	for _, minute := range []int{0, 1, 3, 6} {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	scan, err := store.ScanMarketCandleGaps(ctx, data.MarketCandleGapScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scan.Window.Count != 4 || scan.Window.From == nil || !scan.Window.From.Equal(start) ||
		scan.Window.To == nil || !scan.Window.To.Equal(start.Add(6*time.Minute)) {
		t.Fatalf("unexpected scan window: %#v", scan.Window)
	}
	if scan.Limited || scan.TotalCount != 2 || scan.ReturnedCount != 2 || len(scan.Gaps) != 2 {
		t.Fatalf("unexpected gap metadata: %#v", scan)
	}
	assertTaskGap(t, scan.Gaps[0], start.Add(2*time.Minute), start.Add(3*time.Minute), 1)
	assertTaskGap(t, scan.Gaps[1], start.Add(4*time.Minute), start.Add(6*time.Minute), 2)
}

func TestIntegrationScanMarketCandleGapsReportsHealthyHistory(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCH")
	cleanupMarketCandles(t, store, symbol)
	for _, minute := range []int{0, 1, 2} {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	scan, err := store.ScanMarketCandleGaps(ctx, data.MarketCandleGapScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if scan.Window.Count != 3 || scan.TotalCount != 0 || scan.ReturnedCount != 0 || len(scan.Gaps) != 0 || scan.Limited {
		t.Fatalf("unexpected healthy scan: %#v", scan)
	}
}

func TestIntegrationScanMarketCandleGapsReportsLimitedTotal(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCL")
	cleanupMarketCandles(t, store, symbol)
	for minute := 0; minute <= 10; minute += 2 {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	scan, err := store.ScanMarketCandleGaps(ctx, data.MarketCandleGapScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !scan.Limited || scan.TotalCount != 5 || scan.ReturnedCount != 2 || len(scan.Gaps) != 2 {
		t.Fatalf("unexpected limited scan: %#v", scan)
	}
	assertTaskGap(t, scan.Gaps[0], start.Add(time.Minute), start.Add(2*time.Minute), 1)
	assertTaskGap(t, scan.Gaps[1], start.Add(3*time.Minute), start.Add(4*time.Minute), 1)
}

func TestIntegrationRepairMarketCandleGapCreatesSyncTask(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCR")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})
	for _, minute := range []int{0, 1, 4} {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	request := data.RepairMarketCandleGapRequest{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		From:     start.Add(2 * time.Minute),
		To:       start.Add(4 * time.Minute),
	}
	result, err := store.RepairMarketCandleGap(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != "" || result.SkippedExisting != 0 || result.TotalCount != 1 || result.RepairLimit != 1 || result.Limited {
		t.Fatalf("unexpected repair result metadata: %#v", result)
	}
	if len(result.CreatedTasks) != 1 {
		t.Fatalf("created repair task count = %d, want 1: %#v", len(result.CreatedTasks), result.CreatedTasks)
	}
	assertRepairTaskWindow(t, result.CreatedTasks[0], "", request.From, request.To)

	duplicate, err := store.RepairMarketCandleGap(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(duplicate.CreatedTasks) != 0 || duplicate.SkippedExisting != 1 {
		t.Fatalf("duplicate repair result = %#v, want skipped existing", duplicate)
	}

	notGapRequest := request
	notGapRequest.From = start.Add(time.Minute)
	notGapRequest.To = start.Add(2 * time.Minute)
	if _, err := store.RepairMarketCandleGap(ctx, notGapRequest); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("non-gap repair error = %v, want ErrNotFound", err)
	}
}

func TestIntegrationRepairMarketCandleGapsCreatesSyncTasks(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCB")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})
	for _, minute := range []int{0, 1, 3, 6} {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	request := data.RepairMarketCandleGapsRequest{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Gaps: []data.RepairMarketCandleGapWindow{
			{From: start.Add(2 * time.Minute), To: start.Add(3 * time.Minute)},
			{From: start.Add(4 * time.Minute), To: start.Add(6 * time.Minute)},
		},
	}
	result, err := store.RepairMarketCandleGaps(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != "" || result.SkippedExisting != 0 || result.TotalCount != 2 ||
		result.RepairLimit != data.MaxMarketCandleGapScanLimit || result.Limited {
		t.Fatalf("unexpected batch repair metadata: %#v", result)
	}
	if len(result.CreatedTasks) != 2 {
		t.Fatalf("created repair task count = %d, want 2: %#v", len(result.CreatedTasks), result.CreatedTasks)
	}
	assertRepairTaskWindow(t, result.CreatedTasks[0], "", request.Gaps[0].From, request.Gaps[0].To)
	assertRepairTaskWindow(t, result.CreatedTasks[1], "", request.Gaps[1].From, request.Gaps[1].To)

	duplicate, err := store.RepairMarketCandleGaps(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(duplicate.CreatedTasks) != 0 || duplicate.SkippedExisting != 2 {
		t.Fatalf("duplicate batch repair result = %#v, want skipped existing", duplicate)
	}

	notGapRequest := request
	notGapRequest.Gaps = []data.RepairMarketCandleGapWindow{{From: start.Add(time.Minute), To: start.Add(2 * time.Minute)}}
	if _, err := store.RepairMarketCandleGaps(ctx, notGapRequest); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("non-gap batch repair error = %v, want ErrNotFound", err)
	}
}

func cleanupMarketCandles(t *testing.T, store *Store, symbol string) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})
}

func integrationGapScanCandle(symbol string, start time.Time, minute int) data.Candle {
	openTime := start.Add(time.Duration(minute) * time.Minute)
	return data.Candle{
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  "1m",
		OpenTime:  openTime,
		CloseTime: openTime.Add(time.Minute),
		Open:      "100",
		High:      "101",
		Low:       "99",
		Close:     "100",
		Volume:    "1",
		IsClosed:  true,
	}
}
