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
			assertDataSyncCommandInvalidState(t, err)

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

func TestIntegrationDataSyncRetryRequiresWorkerLeaseOwner(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst_retry_owner")
	symbol := integrationSymbol("RWO")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "owner-worker")
	if _, err := store.pool.Exec(ctx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = 'binance'`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = 'binance'`)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})

	err := store.RecordDataSyncRetry(
		ctx,
		id,
		"other-worker",
		errors.New("binance klines temporary unavailable: api.binance.com: EOF"),
		ptrTime(time.Now().UTC().Add(time.Hour)),
	)
	assertDataSyncCommandInvalidState(t, err)

	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusRunning || !row.syncEnabled || !row.realtimeEnabled {
		t.Fatalf("wrong worker retry changed task state: %#v", row)
	}
	if !row.lockedBy.Valid || row.lockedBy.String != "owner-worker" || !row.lockedUntil.Valid || !row.heartbeatAt.Valid {
		t.Fatalf("wrong worker retry changed lease: %#v", row)
	}
	if row.lastError != "" || row.nextAttemptAt.Valid {
		t.Fatalf("wrong worker retry changed retry fields: %#v", row)
	}

	var backoffCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)::int
		  FROM data_sync_exchange_backoffs
		 WHERE exchange = 'binance'`,
	).Scan(&backoffCount); err != nil {
		t.Fatal(err)
	}
	if backoffCount != 0 {
		t.Fatalf("wrong worker retry wrote %d exchange backoffs, want 0", backoffCount)
	}
}

func TestIntegrationDataSyncFailureRequiresWorkerLeaseOwner(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst_fail_owner")
	symbol := integrationSymbol("FWO")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "owner-worker")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})

	err := store.MarkDataSyncFailed(ctx, id, "other-worker", errors.New("invalid symbol"))
	assertDataSyncCommandInvalidState(t, err)

	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusRunning || !row.syncEnabled || !row.realtimeEnabled {
		t.Fatalf("wrong worker failure changed task state: %#v", row)
	}
	if !row.lockedBy.Valid || row.lockedBy.String != "owner-worker" || !row.lockedUntil.Valid || !row.heartbeatAt.Valid {
		t.Fatalf("wrong worker failure changed lease: %#v", row)
	}
	if row.lastError != "" || row.nextAttemptAt.Valid {
		t.Fatalf("wrong worker failure changed error fields: %#v", row)
	}
}

func TestIntegrationDataSyncReleaseRequiresWorkerLeaseOwner(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst_release_owner")
	symbol := integrationSymbol("LWO")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "owner-worker")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})

	err := store.ReleaseDataSyncTask(ctx, id, "other-worker")
	assertDataSyncCommandInvalidState(t, err)

	row := readIntegrationSyncTask(t, ctx, store, id)
	if !row.lockedBy.Valid || row.lockedBy.String != "owner-worker" || !row.lockedUntil.Valid || !row.heartbeatAt.Valid {
		t.Fatalf("wrong worker release changed lease: %#v", row)
	}
}

func TestIntegrationDataSyncSkippedFetchReleaseRequiresWorkerLeaseOwner(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst_skip_owner")
	symbol := integrationSymbol("SKO")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "owner-worker")
	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET attempt_count = 3
		 WHERE id = $1`,
		id,
	); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})

	err := store.ReleaseDataSyncTaskAfterSkippedFetch(ctx, id, "other-worker")
	assertDataSyncCommandInvalidState(t, err)

	row := readIntegrationSyncTask(t, ctx, store, id)
	if !row.lockedBy.Valid || row.lockedBy.String != "owner-worker" || !row.lockedUntil.Valid || !row.heartbeatAt.Valid {
		t.Fatalf("wrong worker skipped fetch release changed lease: %#v", row)
	}
	var attemptCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT attempt_count
		  FROM data_sync_tasks
		 WHERE id = $1`,
		id,
	).Scan(&attemptCount); err != nil {
		t.Fatal(err)
	}
	if attemptCount != 3 {
		t.Fatalf("wrong worker skipped fetch release changed attempt_count = %d, want 3", attemptCount)
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

func assertDataSyncCommandInvalidState(t *testing.T, err error) {
	t.Helper()

	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("err = %v, want invalid state", err)
	}
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeDataSyncCommandInvalidState {
		t.Fatalf("domain code = %q, %t; want %q, true", code, ok, data.ErrorCodeDataSyncCommandInvalidState)
	}
}
