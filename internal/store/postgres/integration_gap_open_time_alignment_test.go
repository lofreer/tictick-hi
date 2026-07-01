package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationDataSyncTaskGapsIgnoreMisalignedOpenTimeCandles(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 13, 0, 0, 0, time.UTC)
	taskID := integrationID("dst_gapot")
	symbol := integrationSymbol("DGOT")
	misalignedOpenTime := start.Add(time.Minute + 30*time.Second)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})

	insertDataHealthTaskWindow(
		t,
		ctx,
		store,
		taskID,
		symbol,
		data.TaskStatusSucceeded,
		false,
		false,
		&start,
		nil,
		ptrTime(start.Add(3*time.Minute)),
		nil,
		"",
	)
	insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(symbol, start, 0))
	insertLegacyInvalidOpenTimeCandle(t, ctx, store, symbol, misalignedOpenTime)
	insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(symbol, start, 3))

	found := findListedDataSyncTask(t, ctx, store, taskID)
	if found.DataHealth != data.DataSyncHealthInvalid || found.InvalidSummary == nil {
		t.Fatalf("task health = %q invalid=%#v, want invalid health", found.DataHealth, found.InvalidSummary)
	}
	if found.InvalidSummary.FirstIssue == nil ||
		found.InvalidSummary.FirstIssue.Code != data.CandleIssueInvalidOpenTime ||
		found.InvalidSummary.FirstIssue.OpenTime == nil ||
		!found.InvalidSummary.FirstIssue.OpenTime.Equal(misalignedOpenTime) {
		t.Fatalf("unexpected invalid summary: %#v", found.InvalidSummary)
	}
	if found.GapSummary == nil || found.GapSummary.Count != 1 || found.GapSummary.FirstGap == nil {
		t.Fatalf("gap summary = %#v, want one repairable gap", found.GapSummary)
	}
	assertTaskGap(t, *found.GapSummary.FirstGap, start.Add(time.Minute), start.Add(3*time.Minute), 2)

	gapList, err := store.ListDataSyncTaskGaps(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if gapList.TotalCount != 1 || gapList.ReturnedCount != 1 || len(gapList.Gaps) != 1 || gapList.Limited {
		t.Fatalf("unexpected task gap list: %#v", gapList)
	}
	assertTaskGap(t, gapList.Gaps[0], start.Add(time.Minute), start.Add(3*time.Minute), 2)

	bogus := data.RepairDataSyncTaskGapRequest{
		From: start.Add(time.Minute),
		To:   misalignedOpenTime,
	}
	if _, err := store.RepairDataSyncTaskGap(ctx, taskID, bogus); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("bogus misaligned task gap repair err = %v, want ErrNotFound", err)
	}

	result, err := store.RepairDataSyncTaskGaps(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != taskID || result.TotalCount != 1 || result.SkippedExisting != 0 ||
		result.RepairLimit != maxDataSyncGapRepairTasks || result.Limited {
		t.Fatalf("unexpected task gap repair metadata: %#v", result)
	}
	if len(result.CreatedTasks) != 1 {
		t.Fatalf("created task count = %d, want 1: %#v", len(result.CreatedTasks), result.CreatedTasks)
	}
	assertRepairTaskWindow(t, result.CreatedTasks[0], taskID, start.Add(time.Minute), start.Add(3*time.Minute))
}

func TestIntegrationMarketCandleGapsIgnoreMisalignedOpenTimeCandles(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 14, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MGOT")
	misalignedOpenTime := start.Add(time.Minute + 30*time.Second)
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
	insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, 3))

	scan, err := store.ScanMarketCandleGaps(ctx, data.MarketCandleGapScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scan.Window.Count != 2 || scan.Window.From == nil || !scan.Window.From.Equal(start) ||
		scan.Window.To == nil || !scan.Window.To.Equal(start.Add(3*time.Minute)) {
		t.Fatalf("unexpected aligned scan window: %#v", scan.Window)
	}
	if scan.Limited || scan.TotalCount != 1 || scan.ReturnedCount != 1 || len(scan.Gaps) != 1 {
		t.Fatalf("unexpected market gap scan metadata: %#v", scan)
	}
	assertTaskGap(t, scan.Gaps[0], start.Add(time.Minute), start.Add(3*time.Minute), 2)

	bogus := data.RepairMarketCandleGapRequest{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		From:     start.Add(time.Minute),
		To:       misalignedOpenTime,
	}
	if _, err := store.RepairMarketCandleGap(ctx, bogus); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("bogus misaligned market gap repair err = %v, want ErrNotFound", err)
	}

	result, err := store.RepairMarketCandleGap(ctx, data.RepairMarketCandleGapRequest{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		From:     start.Add(time.Minute),
		To:       start.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != "" || result.TotalCount != 1 || result.SkippedExisting != 0 ||
		result.RepairLimit != 1 || result.Limited {
		t.Fatalf("unexpected market repair metadata: %#v", result)
	}
	if len(result.CreatedTasks) != 1 {
		t.Fatalf("created market repair task count = %d, want 1: %#v", len(result.CreatedTasks), result.CreatedTasks)
	}
	assertRepairTaskWindow(t, result.CreatedTasks[0], "", start.Add(time.Minute), start.Add(3*time.Minute))
}
