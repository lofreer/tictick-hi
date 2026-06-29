package postgres

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationDeleteDataSyncTaskSoftDeletesAndKeepsCandles(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst_delete")
	symbol := integrationSymbol("DELETE")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "delete-worker")
	openTime := time.Date(2026, 6, 29, 2, 0, 0, 0, time.UTC)
	insertIntegrationCandle(t, ctx, store, data.Candle{
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
	})
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})

	if err := store.DeleteDataSyncTask(ctx, id); err != nil {
		t.Fatal(err)
	}

	var row struct {
		status          data.TaskStatus
		syncEnabled     bool
		realtimeEnabled bool
		lockedBy        sql.NullString
		lockedUntil     sql.NullTime
		heartbeatAt     sql.NullTime
		deletedAt       sql.NullTime
		finishedAt      sql.NullTime
	}
	if err := store.pool.QueryRow(ctx, `
		SELECT status, sync_enabled, realtime_enabled, locked_by, locked_until,
		       heartbeat_at, deleted_at, finished_at
		  FROM data_sync_tasks
		 WHERE id = $1`,
		id,
	).Scan(
		&row.status,
		&row.syncEnabled,
		&row.realtimeEnabled,
		&row.lockedBy,
		&row.lockedUntil,
		&row.heartbeatAt,
		&row.deletedAt,
		&row.finishedAt,
	); err != nil {
		t.Fatal(err)
	}
	if row.status != data.TaskStatusCancelled || row.syncEnabled || row.realtimeEnabled ||
		row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid ||
		!row.deletedAt.Valid || !row.finishedAt.Valid {
		t.Fatalf("soft-deleted task row = %#v, want cancelled hidden row with cleared lease", row)
	}

	tasks, err := store.ListDataSyncTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if task.ID == id {
			t.Fatalf("soft-deleted task should be hidden from list: %#v", task)
		}
	}

	var candleCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM market_candles
		 WHERE exchange = 'binance'
		   AND symbol = $1
		   AND interval = '1m'`,
		symbol,
	).Scan(&candleCount); err != nil {
		t.Fatal(err)
	}
	if candleCount != 1 {
		t.Fatalf("market candles after task delete = %d, want 1", candleCount)
	}

	_, err = store.SetSyncEnabled(ctx, id, true)
	if !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("start deleted task err = %v, want ErrNotFound", err)
	}
	err = store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID: id,
		Candles: []data.Candle{{
			Exchange:  "binance",
			Symbol:    symbol,
			Interval:  "1m",
			OpenTime:  openTime.Add(time.Minute),
			CloseTime: openTime.Add(2 * time.Minute),
			Open:      "101",
			High:      "102",
			Low:       "100",
			Close:     "101",
			Volume:    "1",
			IsClosed:  true,
		}},
	})
	if !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("save deleted task result err = %v, want ErrNotFound", err)
	}
	if err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM market_candles
		 WHERE exchange = 'binance'
		   AND symbol = $1
		   AND interval = '1m'`,
		symbol,
	).Scan(&candleCount); err != nil {
		t.Fatal(err)
	}
	if candleCount != 1 {
		t.Fatalf("market candles after saving deleted task result = %d, want 1", candleCount)
	}
}
