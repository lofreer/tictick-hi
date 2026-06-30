package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationSaveDataSyncResultRequiresRunningActiveLease(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	cases := []struct {
		name     string
		status   data.TaskStatus
		lockedBy string
		workerID string
		expire   bool
	}{
		{name: "pending task", status: data.TaskStatusPending, workerID: "save-state-worker"},
		{name: "running without lease", status: data.TaskStatusRunning, workerID: "save-state-worker"},
		{name: "running expired lease", status: data.TaskStatusRunning, lockedBy: "stale-save-worker", workerID: "stale-save-worker", expire: true},
		{name: "running different worker lease", status: data.TaskStatusRunning, lockedBy: "owner-save-worker", workerID: "other-save-worker"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			id := integrationID("dst_save_state")
			symbol := integrationSymbol("SAVE")
			openTime := time.Date(2026, 6, 29, 4, 0, 0, 0, time.UTC)
			insertIntegrationSyncTask(t, ctx, store, id, symbol, testCase.status, true, false, testCase.lockedBy)
			t.Cleanup(func() {
				cleanupCtx, cleanupCancel := testContext(t)
				defer cleanupCancel()
				_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
				_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
				_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
			})
			if testCase.expire {
				if _, err := store.pool.Exec(ctx, `
					UPDATE data_sync_tasks
					   SET locked_until = now() - interval '1 second',
					       heartbeat_at = now() - interval '1 minute'
					 WHERE id = $1`,
					id,
				); err != nil {
					t.Fatal(err)
				}
			}

			err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
				TaskID:   id,
				WorkerID: testCase.workerID,
				Candles: []data.Candle{
					integrationResumeCandle(symbol, openTime, "1.5"),
				},
				LastOpenTime: &openTime,
				Completed:    true,
			})
			if !errors.Is(err, data.ErrInvalidState) {
				t.Fatalf("save result err = %v, want invalid state", err)
			}
			if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeDataSyncCommandInvalidState {
				t.Fatalf("save result domain code = %q, %t; want %q, true", code, ok, data.ErrorCodeDataSyncCommandInvalidState)
			}

			var candleCount int
			if err := store.pool.QueryRow(ctx, `
				SELECT count(*)::int
				  FROM market_candles
				 WHERE symbol = $1`,
				symbol,
			).Scan(&candleCount); err != nil {
				t.Fatal(err)
			}
			if candleCount != 0 {
				t.Fatalf("invalid save state wrote %d candles, want 0", candleCount)
			}

			row := readIntegrationSyncTask(t, ctx, store, id)
			if row.status != testCase.status {
				t.Fatalf("invalid save state changed status: %#v", row)
			}
			if row.lastError != "" || row.nextAttemptAt.Valid {
				t.Fatalf("invalid save state changed retry/error fields: %#v", row)
			}
		})
	}
}

func markIntegrationDataSyncTaskRunning(
	t *testing.T,
	ctx context.Context,
	store *Store,
	id string,
	workerID string,
) {
	t.Helper()

	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET status = $2,
		       locked_by = $3,
		       locked_until = now() + interval '1 minute',
		       heartbeat_at = now(),
		       started_at = COALESCE(started_at, now())
		 WHERE id = $1`,
		id,
		data.TaskStatusRunning,
		workerID,
	); err != nil {
		t.Fatal(err)
	}
}
