package postgres

import (
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationClaimDataSyncTaskPrioritizesPendingRepairOverRealtimePoll(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")

	start := time.Date(2026, 6, 27, 6, 0, 0, 0, time.UTC)
	realtimeID := integrationID("dst")
	sourceID := integrationID("dst")
	realtimeSymbol := integrationSymbol("CLR")
	repairSymbol := integrationSymbol("CLP")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol IN ($1, $2)`, realtimeSymbol, repairSymbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, repairSymbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol IN ($1, $2)`, realtimeSymbol, repairSymbol)
	})

	insertIntegrationSyncTask(t, ctx, store, realtimeID, realtimeSymbol, data.TaskStatusRunning, true, true, "")
	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET created_at = '1980-01-01T00:00:00Z'::timestamptz
		 WHERE id = $1`,
		realtimeID,
	); err != nil {
		t.Fatal(err)
	}

	insertDataHealthTaskWindow(t, ctx, store, sourceID, repairSymbol, data.TaskStatusSucceeded, false, false, &start, nil, ptrTime(start.Add(5*time.Minute)), nil, "")
	for _, minute := range []int{0, 1, 5} {
		insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(repairSymbol, start, minute))
	}
	repair, err := store.RepairDataSyncTaskGap(ctx, sourceID, data.RepairDataSyncTaskGapRequest{
		From: start.Add(2 * time.Minute),
		To:   start.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(repair.CreatedTasks) != 1 {
		t.Fatalf("repair created tasks = %d, want 1: %#v", len(repair.CreatedTasks), repair)
	}
	repairTaskID := repair.CreatedTasks[0].ID
	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET created_at = '1990-01-01T00:00:00Z'::timestamptz
		 WHERE id = $1`,
		repairTaskID,
	); err != nil {
		t.Fatal(err)
	}

	claimed, ok, err := store.ClaimDataSyncTask(ctx, "claim-order-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected a claimable repair task")
	}
	if claimed.ID != repairTaskID || claimed.RepairSourceTaskID != sourceID || !claimed.SyncEnabled || claimed.RealtimeEnabled {
		t.Fatalf("claimed task = %#v, want pending repair task %s", claimed, repairTaskID)
	}

	realtime := readIntegrationSyncTask(t, ctx, store, realtimeID)
	if realtime.lockedBy.Valid || realtime.lockedUntil.Valid || realtime.heartbeatAt.Valid {
		t.Fatalf("realtime task should remain unclaimed while repair is pending: %#v", realtime)
	}
}
