package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationEmptyCompletedDataSyncResultStopsOneShotLoop(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 29, 1, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)
	id := integrationID("dst")
	symbol := integrationSymbol("EMPTY")
	insertDataHealthTaskWindow(t, ctx, store, id, symbol, data.TaskStatusRunning, true, false, &start, &end, nil, nil, "")
	markIntegrationDataSyncTaskRunning(t, ctx, store, id, "empty-save-worker")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})

	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{TaskID: id, Completed: true}); err != nil {
		t.Fatal(err)
	}
	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusSucceeded || row.syncEnabled || row.realtimeEnabled {
		t.Fatalf("empty completed result should stop one-shot task: %#v", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("empty completed result should release lease: %#v", row)
	}
	var latestSynced sql.NullTime
	if err := store.pool.QueryRow(ctx, `
		SELECT last_synced_open_time
		  FROM data_sync_tasks
		 WHERE id = $1`,
		id,
	).Scan(&latestSynced); err != nil {
		t.Fatal(err)
	}
	if latestSynced.Valid {
		t.Fatalf("empty completed result should not advance cursor: %#v", latestSynced)
	}
	listed := findListedDataSyncTask(t, ctx, store, id)
	if listed.DataHealth != data.DataSyncHealthGap || listed.GapSummary == nil || listed.GapSummary.Count != 1 {
		t.Fatalf("listed empty window health = %q summary=%#v, want gap", listed.DataHealth, listed.GapSummary)
	}

	restarted, err := store.SetSyncEnabled(ctx, id, true)
	if err != nil {
		t.Fatal(err)
	}
	if restarted.Status != data.TaskStatusPending || !restarted.SyncEnabled || restarted.DataHealth != data.DataSyncHealthSyncing {
		t.Fatalf("restarted task = %#v, want pending sync", restarted)
	}
}
