package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationListDataSyncTasksReportsDataHealth(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")

	start := time.Date(2026, 6, 27, 3, 0, 0, 0, time.UTC)
	taskSymbols := map[string]string{
		"gap":     integrationSymbol("DHG"),
		"ok":      integrationSymbol("DHO"),
		"syncing": integrationSymbol("DHS"),
		"paused":  integrationSymbol("DHP"),
		"retry":   integrationSymbol("DHR"),
		"failed":  integrationSymbol("DHF"),
	}
	taskIDs := map[string]string{}
	for key, symbol := range taskSymbols {
		taskIDs[key] = integrationID("dst")
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := testContext(t)
			defer cleanupCancel()
			_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, taskIDs[key])
			_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		})
	}

	insertDataHealthTask(t, ctx, store, taskIDs["gap"], taskSymbols["gap"], data.TaskStatusSucceeded, false, false, ptrTime(start.Add(6*time.Minute)), nil, "")
	for _, minute := range []int{0, 1, 3, 6} {
		insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(taskSymbols["gap"], start, minute))
	}

	insertDataHealthTask(t, ctx, store, taskIDs["ok"], taskSymbols["ok"], data.TaskStatusSucceeded, false, false, ptrTime(start.Add(2*time.Minute)), nil, "")
	for _, minute := range []int{0, 1, 2} {
		insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(taskSymbols["ok"], start, minute))
	}

	insertDataHealthTask(t, ctx, store, taskIDs["syncing"], taskSymbols["syncing"], data.TaskStatusRunning, true, false, nil, nil, "")
	insertDataHealthTask(t, ctx, store, taskIDs["paused"], taskSymbols["paused"], data.TaskStatusPaused, false, false, nil, nil, "")
	insertDataHealthTask(t, ctx, store, taskIDs["retry"], taskSymbols["retry"], data.TaskStatusRunning, true, false, nil, ptrTime(time.Now().UTC().Add(time.Hour)), "temporary EOF")
	insertDataHealthTask(t, ctx, store, taskIDs["failed"], taskSymbols["failed"], data.TaskStatusFailed, false, false, nil, nil, "invalid symbol")

	tasks, err := store.ListDataSyncTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	healthByID := make(map[string]data.DataSyncHealth)
	gapSummaryByID := make(map[string]*data.DataSyncGapSummary)
	for _, task := range tasks {
		healthByID[task.ID] = task.DataHealth
		gapSummaryByID[task.ID] = task.GapSummary
	}

	expected := map[string]data.DataSyncHealth{
		taskIDs["gap"]:     data.DataSyncHealthGap,
		taskIDs["ok"]:      data.DataSyncHealthOK,
		taskIDs["syncing"]: data.DataSyncHealthSyncing,
		taskIDs["paused"]:  data.DataSyncHealthPaused,
		taskIDs["retry"]:   data.DataSyncHealthRetrying,
		taskIDs["failed"]:  data.DataSyncHealthFailed,
	}
	for id, want := range expected {
		if got := healthByID[id]; got != want {
			t.Fatalf("task %s data health = %q, want %q", id, got, want)
		}
	}

	gapSummary := gapSummaryByID[taskIDs["gap"]]
	if gapSummary == nil {
		t.Fatal("gap task should expose gap summary")
	}
	if gapSummary.Count != 2 {
		t.Fatalf("gap summary count = %d, want 2", gapSummary.Count)
	}
	if gapSummary.FirstGap == nil {
		t.Fatal("gap task should expose first gap")
	}
	if !gapSummary.FirstGap.From.Equal(start.Add(2*time.Minute)) ||
		!gapSummary.FirstGap.To.Equal(start.Add(3*time.Minute)) ||
		gapSummary.FirstGap.MissingCandles != 1 {
		t.Fatalf("unexpected first gap summary: %#v", gapSummary.FirstGap)
	}
	if gapSummaryByID[taskIDs["ok"]] != nil {
		t.Fatalf("ok task gap summary = %#v, want nil", gapSummaryByID[taskIDs["ok"]])
	}
}

func TestIntegrationListDataSyncTasksReportsExchangeBackoff(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")

	taskID := integrationID("dst")
	symbol := integrationSymbol("DHB")
	backoffUntil := time.Now().UTC().Add(time.Hour)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, taskID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = 'binance'`)
	})
	insertDataHealthTask(t, ctx, store, taskID, symbol, data.TaskStatusPending, true, false, nil, nil, "")
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_exchange_backoffs (exchange, next_attempt_at, last_error, updated_at)
		VALUES ('binance', $1, $2, now())`,
		backoffUntil,
		`binance klines temporary unavailable: api.binance.com: Get "https://api.binance.com/api/v3/klines?symbol=BTCUSDT": EOF`,
	); err != nil {
		t.Fatal(err)
	}

	tasks, err := store.ListDataSyncTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var found *data.DataSyncTask
	for index := range tasks {
		if tasks[index].ID == taskID {
			found = &tasks[index]
			break
		}
	}
	if found == nil {
		t.Fatal("backoff task not listed")
	}
	if found.DataHealth != data.DataSyncHealthRetrying {
		t.Fatalf("data health = %q, want retrying", found.DataHealth)
	}
	if found.ExchangeBackoffUntil == nil || found.ExchangeBackoffUntil.Sub(backoffUntil).Abs() > time.Second {
		t.Fatalf("exchange backoff until = %#v, want %s", found.ExchangeBackoffUntil, backoffUntil)
	}
	if found.ExchangeBackoffError == "" {
		t.Fatal("exchange backoff error should be exposed before API sanitization")
	}
}

func TestIntegrationListDataSyncTaskGapsReportsWindows(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 4, 30, 0, 0, time.UTC)
	symbol := integrationSymbol("DHGD")
	taskID := integrationID("dst")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	insertDataHealthTask(t, ctx, store, taskID, symbol, data.TaskStatusSucceeded, false, false, ptrTime(start.Add(6*time.Minute)), nil, "")
	for _, minute := range []int{0, 1, 3, 6} {
		insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(symbol, start, minute))
	}

	gaps, err := store.ListDataSyncTaskGaps(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if gaps.TaskID != taskID || gaps.Limited || len(gaps.Gaps) != 2 {
		t.Fatalf("unexpected gap list metadata: %#v", gaps)
	}
	if gaps.TotalCount != 2 || gaps.ReturnedCount != 2 || gaps.RepairLimit != 20 {
		t.Fatalf("unexpected gap list counts: %#v", gaps)
	}
	assertTaskGap(t, gaps.Gaps[0], start.Add(2*time.Minute), start.Add(3*time.Minute), 1)
	assertTaskGap(t, gaps.Gaps[1], start.Add(4*time.Minute), start.Add(6*time.Minute), 2)

	if _, err := store.ListDataSyncTaskGaps(ctx, integrationID("missing")); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("missing task error = %v, want ErrNotFound", err)
	}
}

func TestIntegrationListDataSyncTaskGapsReportsLimitedTotal(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 5, 30, 0, 0, time.UTC)
	symbol := integrationSymbol("DHLG")
	taskID := integrationID("dst")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	insertDataHealthTask(t, ctx, store, taskID, symbol, data.TaskStatusSucceeded, false, false, ptrTime(start.Add(44*time.Minute)), nil, "")
	for minute := 0; minute <= 44; minute += 2 {
		insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(symbol, start, minute))
	}

	gaps, err := store.ListDataSyncTaskGaps(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if !gaps.Limited || gaps.TotalCount != 22 || gaps.ReturnedCount != 20 || len(gaps.Gaps) != 20 {
		t.Fatalf("unexpected limited gap list metadata: %#v", gaps)
	}
	if gaps.RepairLimit != 20 {
		t.Fatalf("repair limit = %d, want 20", gaps.RepairLimit)
	}
	assertTaskGap(t, gaps.Gaps[0], start.Add(time.Minute), start.Add(2*time.Minute), 1)
	assertTaskGap(t, gaps.Gaps[19], start.Add(39*time.Minute), start.Add(40*time.Minute), 1)
}

func TestIntegrationRepairDataSyncTaskGapsCreatesSyncTasks(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 4, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("DHRP")
	taskID := integrationID("dst")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	insertDataHealthTask(t, ctx, store, taskID, symbol, data.TaskStatusSucceeded, false, false, ptrTime(start.Add(6*time.Minute)), nil, "")
	for _, minute := range []int{0, 1, 3, 6} {
		insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(symbol, start, minute))
	}

	result, err := store.RepairDataSyncTaskGaps(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != taskID || result.SkippedExisting != 0 || result.Limited {
		t.Fatalf("unexpected repair result metadata: %#v", result)
	}
	if result.TotalCount != 2 || result.RepairLimit != 20 {
		t.Fatalf("unexpected repair result counts: %#v", result)
	}
	if len(result.CreatedTasks) != 2 {
		t.Fatalf("created repair task count = %d, want 2: %#v", len(result.CreatedTasks), result.CreatedTasks)
	}
	assertRepairTaskWindow(t, result.CreatedTasks[0], taskID, start.Add(2*time.Minute), start.Add(3*time.Minute))
	assertRepairTaskWindow(t, result.CreatedTasks[1], taskID, start.Add(4*time.Minute), start.Add(6*time.Minute))

	tasks, err := store.ListDataSyncTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repairSourceByID := make(map[string]string)
	for _, task := range tasks {
		repairSourceByID[task.ID] = task.RepairSourceTaskID
	}
	for _, task := range result.CreatedTasks {
		if repairSourceByID[task.ID] != taskID {
			t.Fatalf("listed repair source for %s = %q, want %q", task.ID, repairSourceByID[task.ID], taskID)
		}
	}

	duplicateResult, err := store.RepairDataSyncTaskGaps(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if len(duplicateResult.CreatedTasks) != 0 || duplicateResult.SkippedExisting != 2 {
		t.Fatalf("duplicate repair result = %#v, want skipped existing", duplicateResult)
	}
}

func TestIntegrationRepairDataSyncTaskGapCreatesSyncTask(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 5, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("DSGP")
	taskID := integrationID("dst")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
	})

	insertDataHealthTask(t, ctx, store, taskID, symbol, data.TaskStatusSucceeded, false, false, ptrTime(start.Add(10*time.Minute)), nil, "")
	request := data.RepairDataSyncTaskGapRequest{
		From: start.Add(2 * time.Minute),
		To:   start.Add(5 * time.Minute),
	}
	result, err := store.RepairDataSyncTaskGap(ctx, taskID, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != taskID || result.SkippedExisting != 0 || result.Limited {
		t.Fatalf("unexpected single repair metadata: %#v", result)
	}
	if result.TotalCount != 1 || result.RepairLimit != 1 || len(result.CreatedTasks) != 1 {
		t.Fatalf("unexpected single repair result: %#v", result)
	}
	assertRepairTaskWindow(t, result.CreatedTasks[0], taskID, request.From, request.To)

	duplicateResult, err := store.RepairDataSyncTaskGap(ctx, taskID, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(duplicateResult.CreatedTasks) != 0 || duplicateResult.SkippedExisting != 1 {
		t.Fatalf("duplicate single repair result = %#v, want skipped existing", duplicateResult)
	}
}

func assertTaskGap(t *testing.T, gap data.CandleGap, from time.Time, to time.Time, missingCandles int) {
	t.Helper()
	if !gap.From.Equal(from) || !gap.To.Equal(to) || gap.MissingCandles != missingCandles {
		t.Fatalf("unexpected task gap: %#v", gap)
	}
}

func assertRepairTaskWindow(t *testing.T, task data.DataSyncTask, sourceTaskID string, from time.Time, to time.Time) {
	t.Helper()
	if task.StartTime == nil || !task.StartTime.Equal(from) ||
		task.EndTime == nil || !task.EndTime.Equal(to) ||
		task.RepairSourceTaskID != sourceTaskID ||
		!task.SyncEnabled ||
		task.RealtimeEnabled ||
		task.Status != data.TaskStatusPending ||
		task.DataHealth != data.DataSyncHealthSyncing {
		t.Fatalf("unexpected repair task: %#v", task)
	}
}

func integrationDataHealthCandle(symbol string, start time.Time, minute int) data.Candle {
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

func insertDataHealthTask(
	t *testing.T,
	ctx context.Context,
	store *Store,
	id string,
	symbol string,
	status data.TaskStatus,
	syncEnabled bool,
	realtimeEnabled bool,
	latestSyncedAt *time.Time,
	nextAttemptAt *time.Time,
	lastError string,
) {
	t.Helper()

	var finishedAt any
	if status == data.TaskStatusSucceeded || status == data.TaskStatusFailed || status == data.TaskStatusCancelled {
		finishedAt = time.Now().UTC()
	}

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_tasks (
			id, exchange, symbol, interval, start_time, sync_enabled, realtime_enabled, status,
			last_synced_open_time, next_attempt_at, last_error, finished_at, created_at, updated_at
		)
		VALUES ($1, 'binance', $2, '1m', $3, $4, $5, $6, $7, $8, NULLIF($9, ''),
		        $10, '2000-01-01T00:00:00Z'::timestamptz, now())`,
		id,
		symbol,
		time.Date(2026, 6, 27, 3, 0, 0, 0, time.UTC),
		syncEnabled,
		realtimeEnabled,
		status,
		latestSyncedAt,
		nextAttemptAt,
		lastError,
		finishedAt,
	); err != nil {
		t.Fatal(err)
	}
}

func clearIntegrationExchangeBackoff(t *testing.T, ctx context.Context, store *Store, exchange string) {
	t.Helper()
	if _, err := store.pool.Exec(ctx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = $1`, exchange); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = $1`, exchange)
	})
}
