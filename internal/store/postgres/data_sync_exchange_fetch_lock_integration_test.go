package postgres

import (
	"database/sql"
	"testing"
	"time"
)

func TestIntegrationDataSyncExchangeFetchLockIsExclusivePerExchange(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	unlockBinance, locked, err := store.TryLockDataSyncExchangeFetch(ctx, "binance")
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("first binance fetch lock was not acquired")
	}
	defer func() {
		if unlockBinance != nil {
			if err := unlockBinance(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}()

	unlockSecondBinance, locked, err := store.TryLockDataSyncExchangeFetch(ctx, "binance")
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		if unlockSecondBinance != nil {
			_ = unlockSecondBinance(ctx)
		}
		t.Fatal("second binance fetch lock was acquired while first lock was held")
	}
	if unlockSecondBinance != nil {
		t.Fatal("second binance fetch unlock should be nil when lock is not acquired")
	}

	unlockOKX, locked, err := store.TryLockDataSyncExchangeFetch(ctx, "okx")
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("okx fetch lock was not acquired while binance fetch lock was held")
	}
	if err := unlockOKX(ctx); err != nil {
		t.Fatal(err)
	}

	unlockCatalogBinance, locked, err := store.TryLockMarketInstrumentSync(ctx, "binance")
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("catalog sync lock should not be blocked by data sync fetch lock")
	}
	if err := unlockCatalogBinance(ctx); err != nil {
		t.Fatal(err)
	}

	if err := unlockBinance(ctx); err != nil {
		t.Fatal(err)
	}
	unlockBinance = nil

	unlockAfterRelease, locked, err := store.TryLockDataSyncExchangeFetch(ctx, "binance")
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("binance fetch lock was not reacquired after release")
	}
	if err := unlockAfterRelease(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationReleaseDataSyncTaskAfterSkippedFetchRevertsClaimAttempt(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	taskID := integrationID("dst")
	symbol := integrationSymbol("FL")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, taskID)
	})

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_tasks (
			id, exchange, symbol, interval, sync_enabled, status,
			locked_by, locked_until, heartbeat_at, attempt_count
		)
		VALUES ($1, 'binance', $2, '1m', true, 'running', 'worker-1', now() + interval '1 minute', now(), 3)`,
		taskID,
		symbol,
	); err != nil {
		t.Fatal(err)
	}

	if err := store.ReleaseDataSyncTaskAfterSkippedFetch(ctx, taskID); err != nil {
		t.Fatal(err)
	}

	var lockedBy sql.NullString
	var lockedUntil sql.NullString
	var heartbeatAt sql.NullString
	var attemptCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT locked_by::text, locked_until::text, heartbeat_at::text, attempt_count
		  FROM data_sync_tasks
		 WHERE id = $1`,
		taskID,
	).Scan(&lockedBy, &lockedUntil, &heartbeatAt, &attemptCount); err != nil {
		t.Fatal(err)
	}
	if lockedBy.Valid || lockedUntil.Valid || heartbeatAt.Valid {
		t.Fatalf("lease was not cleared: lockedBy=%v lockedUntil=%v heartbeatAt=%v", lockedBy, lockedUntil, heartbeatAt)
	}
	if attemptCount != 2 {
		t.Fatalf("attempt_count = %d, want 2", attemptCount)
	}
}

func TestIntegrationRecordDataSyncExchangeFetchLockSkippedExposesHealth(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_fetch_lock_skips`)
	})
	if _, err := store.pool.Exec(ctx, `DELETE FROM data_sync_exchange_fetch_lock_skips`); err != nil {
		t.Fatal(err)
	}

	older := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)
	newer := older.Add(5 * time.Minute)
	if err := store.RecordDataSyncExchangeFetchLockSkipped(ctx, "okx", newer); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordDataSyncExchangeFetchLockSkipped(ctx, "okx", older); err != nil {
		t.Fatal(err)
	}

	var skipCount int64
	var lastSkipped time.Time
	if err := store.pool.QueryRow(ctx, `
		SELECT skip_count, last_skipped_at
		  FROM data_sync_exchange_fetch_lock_skips
		 WHERE exchange = 'okx'`,
	).Scan(&skipCount, &lastSkipped); err != nil {
		t.Fatal(err)
	}
	if skipCount != 2 {
		t.Fatalf("skip_count = %d, want 2", skipCount)
	}
	if !lastSkipped.Equal(newer) {
		t.Fatalf("last_skipped_at = %s, want %s", lastSkipped, newer)
	}

	health, err := store.SystemHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	syncHealth := findIntegrationServiceHealth(health, "sync-worker")
	if syncHealth.FetchLockSkipCount == nil || *syncHealth.FetchLockSkipCount != 2 {
		t.Fatalf("system health fetch lock skip count = %#v, want 2", syncHealth)
	}
	if syncHealth.LastFetchLockSkippedAt == nil || !syncHealth.LastFetchLockSkippedAt.Equal(newer) {
		t.Fatalf("system health last fetch lock skip = %#v, want %s", syncHealth, newer)
	}
	if syncHealth.Status != "ok" {
		t.Fatalf("fetch lock skip metric should not degrade service by itself: %#v", syncHealth)
	}
}
