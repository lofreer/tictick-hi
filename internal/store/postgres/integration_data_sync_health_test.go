package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationListDataSyncTasksReportsDataHealth(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

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

	insertDataHealthTask(t, ctx, store, taskIDs["gap"], taskSymbols["gap"], data.TaskStatusSucceeded, false, false, ptrTime(start.Add(3*time.Minute)), nil, "")
	for _, minute := range []int{0, 1, 3} {
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
	for _, task := range tasks {
		healthByID[task.ID] = task.DataHealth
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
