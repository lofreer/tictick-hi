package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationScanMarketCandleInvalidIssuesReportsPersistedHistory(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 7, 30, 0, 0, time.UTC)
	symbol := integrationSymbol("MCI")
	cleanupMarketCandles(t, store, symbol)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		ensurePositivePriceConstraint(t, cleanupCtx, store)
	})
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 0))
	insertLegacyInvalidDataHealthCandle(t, ctx, store, symbol, start.Add(time.Minute))
	insertLegacyInvalidCloseDataHealthCandle(t, ctx, store, symbol, start.Add(2*time.Minute))
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 3))

	scan, err := store.ScanMarketCandleInvalidIssues(ctx, data.MarketCandleInvalidIssueScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scan.Window.Count != 4 || scan.Window.From == nil || !scan.Window.From.Equal(start) ||
		scan.Window.To == nil || !scan.Window.To.Equal(start.Add(3*time.Minute)) {
		t.Fatalf("unexpected invalid scan window: %#v", scan.Window)
	}
	if !scan.Limited || scan.TotalCount != 2 || scan.ReturnedCount != 1 || len(scan.Issues) != 1 {
		t.Fatalf("unexpected invalid scan metadata: %#v", scan)
	}
	issue := scan.Issues[0]
	if issue.OpenTime == nil || !issue.OpenTime.Equal(start.Add(time.Minute)) ||
		issue.Code != data.CandleIssueInvalidOpenPrice ||
		issue.Message != "open price value must be positive" {
		t.Fatalf("unexpected first invalid issue: %#v", issue)
	}
}

func TestIntegrationScanMarketCandleInvalidIssuesReportsTimeBoundaries(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 7, 35, 0, 0, time.UTC)
	symbol := integrationSymbol("MCIT")
	misalignedOpenTime := start.Add(time.Minute + 30*time.Second)
	closeMismatchOpenTime := start.Add(3 * time.Minute)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})
	upsertIntegrationMarketInstrument(t, ctx, store, "binance", symbol, "active")
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 0))
	insertLegacyInvalidOpenTimeCandle(t, ctx, store, symbol, misalignedOpenTime)
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 2))
	insertLegacyInvalidCloseTimeCandle(t, ctx, store, symbol, closeMismatchOpenTime)

	scan, err := store.ScanMarketCandleInvalidIssues(ctx, data.MarketCandleInvalidIssueScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scan.TotalCount != 2 || scan.ReturnedCount != 2 || len(scan.Issues) != 2 || scan.Limited {
		t.Fatalf("unexpected time invalid scan metadata: %#v", scan)
	}
	if scan.Issues[0].Code != data.CandleIssueInvalidOpenTime ||
		scan.Issues[0].OpenTime == nil ||
		!scan.Issues[0].OpenTime.Equal(misalignedOpenTime) ||
		scan.Issues[1].Code != data.CandleIssueInvalidCloseTime ||
		scan.Issues[1].OpenTime == nil ||
		!scan.Issues[1].OpenTime.Equal(closeMismatchOpenTime) {
		t.Fatalf("unexpected time invalid issues: %#v", scan.Issues)
	}

	repair, err := store.RepairMarketCandleInvalidIssues(ctx, data.RepairMarketCandleInvalidIssuesRequest{
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  "1m",
		OpenTimes: []time.Time{misalignedOpenTime, closeMismatchOpenTime},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repair.TotalCount != 2 || repair.SkippedExisting != 0 || len(repair.CreatedTasks) != 1 {
		t.Fatalf("unexpected time invalid repair result: %#v", repair)
	}
	assertRepairTaskWindow(t, repair.CreatedTasks[0], "", closeMismatchOpenTime, closeMismatchOpenTime.Add(time.Minute))
}

func TestIntegrationScanMarketCandleInvalidIssuesReportsHealthyHistory(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 7, 45, 0, 0, time.UTC)
	symbol := integrationSymbol("MCIH")
	cleanupMarketCandles(t, store, symbol)
	for _, minute := range []int{0, 1, 2} {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	scan, err := store.ScanMarketCandleInvalidIssues(ctx, data.MarketCandleInvalidIssueScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if scan.Window.Count != 3 || scan.TotalCount != 0 || scan.ReturnedCount != 0 || len(scan.Issues) != 0 || scan.Limited {
		t.Fatalf("unexpected healthy invalid scan: %#v", scan)
	}
}

func TestIntegrationRepairMarketCandleInvalidIssuesCreatesSyncTasks(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 8, 30, 0, 0, time.UTC)
	symbol := integrationSymbol("MCIR")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
		ensurePositivePriceConstraint(t, cleanupCtx, store)
	})
	upsertIntegrationMarketInstrument(t, ctx, store, "binance", symbol, "active")
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 0))
	insertLegacyInvalidDataHealthCandle(t, ctx, store, symbol, start.Add(time.Minute))
	insertLegacyInvalidCloseDataHealthCandle(t, ctx, store, symbol, start.Add(2*time.Minute))
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 3))

	request := data.RepairMarketCandleInvalidIssuesRequest{
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  "1m",
		OpenTimes: []time.Time{start.Add(time.Minute), start.Add(2 * time.Minute)},
	}
	result, err := store.RepairMarketCandleInvalidIssues(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != "" || result.SkippedExisting != 0 || result.TotalCount != 2 ||
		result.RepairLimit != data.MaxMarketCandleInvalidIssueScanLimit || result.Limited {
		t.Fatalf("unexpected invalid repair metadata: %#v", result)
	}
	if len(result.CreatedTasks) != 2 {
		t.Fatalf("created repair task count = %d, want 2: %#v", len(result.CreatedTasks), result.CreatedTasks)
	}
	assertRepairTaskWindow(t, result.CreatedTasks[0], "", start.Add(time.Minute), start.Add(2*time.Minute))
	assertRepairTaskWindow(t, result.CreatedTasks[1], "", start.Add(2*time.Minute), start.Add(3*time.Minute))

	duplicate, err := store.RepairMarketCandleInvalidIssues(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(duplicate.CreatedTasks) != 0 || duplicate.SkippedExisting != 2 {
		t.Fatalf("duplicate invalid repair result = %#v, want skipped existing", duplicate)
	}

	notInvalidRequest := request
	notInvalidRequest.OpenTimes = []time.Time{start}
	if _, err := store.RepairMarketCandleInvalidIssues(ctx, notInvalidRequest); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("non-invalid repair error = %v, want ErrNotFound", err)
	}
}

func TestIntegrationRepairMarketCandleInvalidIssueConvergesFullHistoryScan(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 8, 45, 0, 0, time.UTC)
	symbol := integrationSymbol("MCIC")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
		ensurePositivePriceConstraint(t, cleanupCtx, store)
	})
	upsertIntegrationMarketInstrument(t, ctx, store, "binance", symbol, "active")
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 0))
	insertLegacyInvalidDataHealthCandle(t, ctx, store, symbol, start.Add(time.Minute))
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 2))

	before, err := store.ScanMarketCandleInvalidIssues(ctx, data.MarketCandleInvalidIssueScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if before.TotalCount != 1 || before.ReturnedCount != 1 || len(before.Issues) != 1 {
		t.Fatalf("invalid scan before repair = %#v, want one issue", before)
	}

	repair, err := store.RepairMarketCandleInvalidIssues(ctx, data.RepairMarketCandleInvalidIssuesRequest{
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  "1m",
		OpenTimes: []time.Time{start.Add(time.Minute)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repair.TotalCount != 1 || repair.SkippedExisting != 0 || len(repair.CreatedTasks) != 1 {
		t.Fatalf("unexpected invalid repair result: %#v", repair)
	}
	repairTask := repair.CreatedTasks[0]
	assertRepairTaskWindow(t, repairTask, "", start.Add(time.Minute), start.Add(2*time.Minute))

	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET status = $2,
		       locked_by = 'invalid-repair-converge-worker',
		       locked_until = now() + interval '1 minute',
		       heartbeat_at = now()
		 WHERE id = $1`,
		repairTask.ID,
		data.TaskStatusRunning,
	); err != nil {
		t.Fatal(err)
	}

	repairedCandle := integrationGapScanCandle(symbol, start, 1)
	lastOpenTime := repairedCandle.OpenTime
	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:       repairTask.ID,
		WorkerID:     "invalid-repair-converge-worker",
		Candles:      []data.Candle{repairedCandle},
		LastOpenTime: &lastOpenTime,
		Completed:    true,
	}); err != nil {
		t.Fatal(err)
	}

	listedRepair := findListedDataSyncTask(t, ctx, store, repairTask.ID)
	if listedRepair.Status != data.TaskStatusSucceeded ||
		listedRepair.SyncEnabled ||
		listedRepair.LatestSyncedOpenTime == nil ||
		!listedRepair.LatestSyncedOpenTime.Equal(lastOpenTime) {
		t.Fatalf("repair task after result = %#v, want succeeded with latest synced open time", listedRepair)
	}

	after, err := store.ScanMarketCandleInvalidIssues(ctx, data.MarketCandleInvalidIssueScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if after.Window.Count != 3 || after.TotalCount != 0 || after.ReturnedCount != 0 ||
		len(after.Issues) != 0 || after.Limited {
		t.Fatalf("invalid scan after repair result = %#v, want healthy history", after)
	}
}

func TestIntegrationQuarantineMarketCandleInvalidOpenTimeIssues(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 9, 15, 0, 0, time.UTC)
	misaligned := start.Add(90 * time.Second)
	symbol := integrationSymbol("MCIQ")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candle_quarantines WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 0))
	insertLegacyInvalidOpenTimeCandle(t, ctx, store, symbol, misaligned)
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 2))

	result, err := store.QuarantineMarketCandleInvalidIssues(ctx, data.QuarantineMarketCandleInvalidIssuesRequest{
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  "1m",
		OpenTimes: []time.Time{misaligned},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCount != 1 || result.SkippedNonQuarantinable != 0 || len(result.Quarantined) != 1 {
		t.Fatalf("unexpected quarantine result: %#v", result)
	}
	if result.Quarantined[0].Reason != data.CandleIssueInvalidOpenTime ||
		!result.Quarantined[0].OpenTime.Equal(misaligned) {
		t.Fatalf("unexpected quarantine record: %#v", result.Quarantined[0])
	}

	var activeCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		  FROM market_candles
		 WHERE exchange = 'binance'
		   AND symbol = $1
		   AND interval = '1m'
		   AND open_time = $2`,
		symbol,
		misaligned,
	).Scan(&activeCount); err != nil {
		t.Fatal(err)
	}
	if activeCount != 0 {
		t.Fatalf("active misaligned candle count = %d, want 0", activeCount)
	}

	var archivedReason string
	if err := store.pool.QueryRow(ctx, `
		SELECT reason
		  FROM market_candle_quarantines
		 WHERE exchange = 'binance'
		   AND symbol = $1
		   AND interval = '1m'
		   AND open_time = $2`,
		symbol,
		misaligned,
	).Scan(&archivedReason); err != nil {
		t.Fatal(err)
	}
	if archivedReason != data.CandleIssueInvalidOpenTime {
		t.Fatalf("archived reason = %q, want invalid open time", archivedReason)
	}

	scan, err := store.ScanMarketCandleInvalidIssues(ctx, data.MarketCandleInvalidIssueScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if scan.TotalCount != 0 || len(scan.Issues) != 0 {
		t.Fatalf("invalid scan after quarantine = %#v, want no active invalid issues", scan)
	}
}
