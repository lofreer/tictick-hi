package postgres

import (
	"database/sql"
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
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})
	upsertIntegrationMarketInstrument(t, ctx, store, "binance", symbol, "active")
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

func TestIntegrationMarketCandleRepairsRejectUnsupportedDataSyncInterval(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 9, 30, 0, 0, time.UTC)
	symbol := integrationSymbol("MRI")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
	})
	cases := []struct {
		name string
		run  func() error
	}{
		{name: "single gap", run: func() error {
			_, err := store.RepairMarketCandleGap(ctx, data.RepairMarketCandleGapRequest{
				Exchange: "binance", Symbol: symbol, Interval: "2m", From: start, To: start.Add(2 * time.Minute),
			})
			return err
		}},
		{name: "batch gaps", run: func() error {
			_, err := store.RepairMarketCandleGaps(ctx, data.RepairMarketCandleGapsRequest{
				Exchange: "binance", Symbol: symbol, Interval: "2m", Gaps: []data.RepairMarketCandleGapWindow{{From: start, To: start.Add(2 * time.Minute)}},
			})
			return err
		}},
		{name: "invalid issues", run: func() error {
			_, err := store.RepairMarketCandleInvalidIssues(ctx, data.RepairMarketCandleInvalidIssuesRequest{
				Exchange: "binance", Symbol: symbol, Interval: "2m", OpenTimes: []time.Time{start},
			})
			return err
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.run()
			if err == nil || err.Error() != `unsupported data sync interval "2m"` {
				t.Fatalf("repair err = %v, want unsupported interval", err)
			}
		})
	}
	var taskCount int
	if err := store.pool.QueryRow(ctx, `SELECT count(*)::int FROM data_sync_tasks WHERE symbol = $1`, symbol).Scan(&taskCount); err != nil {
		t.Fatal(err)
	}
	if taskCount != 0 {
		t.Fatalf("unsupported market repair created %d tasks, want 0", taskCount)
	}
}

func TestIntegrationRepairMarketCandleGapIgnoresSoftDeletedRepairTask(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCD")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})
	upsertIntegrationMarketInstrument(t, ctx, store, "binance", symbol, "active")
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
	first, err := store.RepairMarketCandleGap(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.CreatedTasks) != 1 {
		t.Fatalf("first repair created task count = %d, want 1: %#v", len(first.CreatedTasks), first.CreatedTasks)
	}
	firstTaskID := first.CreatedTasks[0].ID
	if err := store.DeleteDataSyncTask(ctx, firstTaskID); err != nil {
		t.Fatal(err)
	}

	var deletedStatus data.TaskStatus
	var deletedAt sql.NullTime
	if err := store.pool.QueryRow(ctx, `
		SELECT status, deleted_at
		  FROM data_sync_tasks
		 WHERE id = $1`,
		firstTaskID,
	).Scan(&deletedStatus, &deletedAt); err != nil {
		t.Fatal(err)
	}
	if deletedStatus != data.TaskStatusCancelled || !deletedAt.Valid {
		t.Fatalf("deleted repair task status/deleted_at = %s/%#v, want cancelled with deleted_at", deletedStatus, deletedAt)
	}

	second, err := store.RepairMarketCandleGap(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if second.SkippedExisting != 0 || len(second.CreatedTasks) != 1 {
		t.Fatalf("second repair result = %#v, want new task after soft delete", second)
	}
	if second.CreatedTasks[0].ID == firstTaskID {
		t.Fatalf("second repair reused soft-deleted task id %s", firstTaskID)
	}
	assertRepairTaskWindow(t, second.CreatedTasks[0], "", request.From, request.To)

	var activeCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM data_sync_tasks
		 WHERE symbol = $1
		   AND start_time = $2
		   AND end_time = $3
		   AND deleted_at IS NULL`,
		symbol,
		request.From,
		request.To,
	).Scan(&activeCount); err != nil {
		t.Fatal(err)
	}
	if activeCount != 1 {
		t.Fatalf("active repair task count = %d, want 1", activeCount)
	}

	var totalCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM data_sync_tasks
		 WHERE symbol = $1
		   AND start_time = $2
		   AND end_time = $3`,
		symbol,
		request.From,
		request.To,
	).Scan(&totalCount); err != nil {
		t.Fatal(err)
	}
	if totalCount != 2 {
		t.Fatalf("total repair task count = %d, want retained soft-deleted task plus new active task", totalCount)
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
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})
	upsertIntegrationMarketInstrument(t, ctx, store, "binance", symbol, "active")
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

func TestIntegrationRepairMarketCandleGapsRollsBackWhenAnyGapIsInvalid(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCA")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})
	upsertIntegrationMarketInstrument(t, ctx, store, "binance", symbol, "active")
	for _, minute := range []int{0, 1, 3, 6} {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	request := data.RepairMarketCandleGapsRequest{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Gaps: []data.RepairMarketCandleGapWindow{
			{From: start.Add(2 * time.Minute), To: start.Add(3 * time.Minute)},
			{From: start.Add(time.Minute), To: start.Add(2 * time.Minute)},
		},
	}
	if _, err := store.RepairMarketCandleGaps(ctx, request); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("mixed valid/invalid batch repair error = %v, want ErrNotFound", err)
	}

	var taskCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM data_sync_tasks
		 WHERE symbol = $1`,
		symbol,
	).Scan(&taskCount); err != nil {
		t.Fatal(err)
	}
	if taskCount != 0 {
		t.Fatalf("repair task count after rolled back batch = %d, want 0", taskCount)
	}

	single, err := store.RepairMarketCandleGap(ctx, data.RepairMarketCandleGapRequest{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		From:     request.Gaps[0].From,
		To:       request.Gaps[0].To,
	})
	if err != nil {
		t.Fatal(err)
	}
	if single.SkippedExisting != 0 || len(single.CreatedTasks) != 1 {
		t.Fatalf("single repair after rollback = %#v, want one newly created task", single)
	}
	assertRepairTaskWindow(t, single.CreatedTasks[0], "", request.Gaps[0].From, request.Gaps[0].To)
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
