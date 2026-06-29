package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationRepairTaskExecutionConvergesSourceDataHealth(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")

	start := time.Date(2026, 6, 27, 5, 30, 0, 0, time.UTC)
	symbol := integrationSymbol("DSRC")
	sourceID := integrationID("dst")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	insertDataHealthTaskWindow(t, ctx, store, sourceID, symbol, data.TaskStatusSucceeded, false, false, &start, nil, ptrTime(start.Add(5*time.Minute)), nil, "")
	for _, minute := range []int{0, 1, 5} {
		insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(symbol, start, minute))
	}

	before := findListedDataSyncTask(t, ctx, store, sourceID)
	if before.DataHealth != data.DataSyncHealthGap || before.GapSummary == nil || before.GapSummary.Count != 1 {
		t.Fatalf("source health before repair = %q summary=%#v, want one gap", before.DataHealth, before.GapSummary)
	}

	request := data.RepairDataSyncTaskGapRequest{
		From: start.Add(2 * time.Minute),
		To:   start.Add(5 * time.Minute),
	}
	repair, err := store.RepairDataSyncTaskGap(ctx, sourceID, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(repair.CreatedTasks) != 1 {
		t.Fatalf("repair created tasks = %d, want 1: %#v", len(repair.CreatedTasks), repair)
	}
	repairTask := repair.CreatedTasks[0]
	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET status = $2,
		       locked_by = 'repair-converge-worker',
		       locked_until = now() + interval '1 minute',
		       heartbeat_at = now(),
		       created_at = '1999-01-01T00:00:00Z'::timestamptz
		 WHERE id = $1`,
		repairTask.ID,
		data.TaskStatusRunning,
	); err != nil {
		t.Fatal(err)
	}

	lastRepairOpenTime := start.Add(4 * time.Minute)
	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID: repairTask.ID,
		Candles: []data.Candle{
			integrationDataHealthCandle(symbol, start, 2),
			integrationDataHealthCandle(symbol, start, 3),
			integrationDataHealthCandle(symbol, start, 4),
		},
		LastOpenTime: &lastRepairOpenTime,
		Completed:    true,
	}); err != nil {
		t.Fatal(err)
	}

	listedRepair := findListedDataSyncTask(t, ctx, store, repairTask.ID)
	if listedRepair.Status != data.TaskStatusSucceeded || listedRepair.SyncEnabled ||
		listedRepair.LatestSyncedOpenTime == nil ||
		!listedRepair.LatestSyncedOpenTime.Equal(lastRepairOpenTime) {
		t.Fatalf("repair task after saved result = %#v", listedRepair)
	}

	converged := findListedDataSyncTask(t, ctx, store, sourceID)
	if converged.DataHealth != data.DataSyncHealthOK || converged.GapSummary != nil {
		t.Fatalf("source health after repair result = %q summary=%#v, want ok with no gap", converged.DataHealth, converged.GapSummary)
	}
}

func findListedDataSyncTask(t *testing.T, ctx context.Context, store *Store, id string) data.DataSyncTask {
	t.Helper()

	tasks, err := store.ListDataSyncTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if task.ID == id {
			return task
		}
	}
	t.Fatalf("data sync task %s not found in list", id)
	return data.DataSyncTask{}
}
