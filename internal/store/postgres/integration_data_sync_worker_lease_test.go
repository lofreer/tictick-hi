package postgres

import (
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationListDataSyncTasksReportsWorkerLease(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	taskID := integrationID("dst")
	symbol := integrationSymbol("DHL")
	lockedUntil := time.Now().UTC().Add(2 * time.Minute)
	heartbeatAt := time.Now().UTC().Add(-15 * time.Second)
	startedAt := time.Now().UTC().Add(-time.Minute)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, taskID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})
	insertDataHealthTask(t, ctx, store, taskID, symbol, data.TaskStatusRunning, true, false, nil, nil, "")
	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET locked_by = $2,
		       locked_until = $3,
		       heartbeat_at = $4,
		       started_at = $5
		 WHERE id = $1`,
		taskID,
		"sync-worker-1",
		lockedUntil,
		heartbeatAt,
		startedAt,
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
		t.Fatal("leased task not listed")
	}
	assertIntegrationDataSyncWorkerLease(t, *found, lockedUntil, heartbeatAt, startedAt)

	direct, err := store.GetDataSyncTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	assertIntegrationDataSyncWorkerLease(t, direct, lockedUntil, heartbeatAt, startedAt)
}

func assertIntegrationDataSyncWorkerLease(t *testing.T, task data.DataSyncTask, lockedUntil time.Time, heartbeatAt time.Time, startedAt time.Time) {
	t.Helper()
	if task.LockedBy != "sync-worker-1" {
		t.Fatalf("locked by = %q, want sync-worker-1", task.LockedBy)
	}
	if task.LockedUntil == nil || task.LockedUntil.Sub(lockedUntil).Abs() > time.Second {
		t.Fatalf("locked until = %#v, want %s", task.LockedUntil, lockedUntil)
	}
	if task.HeartbeatAt == nil || task.HeartbeatAt.Sub(heartbeatAt).Abs() > time.Second {
		t.Fatalf("heartbeat at = %#v, want %s", task.HeartbeatAt, heartbeatAt)
	}
	if task.StartedAt == nil || task.StartedAt.Sub(startedAt).Abs() > time.Second {
		t.Fatalf("started at = %#v, want %s", task.StartedAt, startedAt)
	}
	if task.FinishedAt != nil {
		t.Fatalf("finished at = %#v, want nil for running task", task.FinishedAt)
	}
}
