package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationListDataSyncTasksReportsInvalidCandleHealth(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)
	taskID := integrationID("dst")
	symbol := integrationSymbol("DHI")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, taskID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		ensurePositivePriceConstraint(t, cleanupCtx, store)
	})

	insertDataHealthTaskWindow(
		t,
		ctx,
		store,
		taskID,
		symbol,
		data.TaskStatusSucceeded,
		false,
		false,
		&start,
		nil,
		ptrTime(start.Add(2*time.Minute)),
		nil,
		"",
	)
	for _, minute := range []int{0, 1} {
		insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(symbol, start, minute))
	}
	insertLegacyInvalidDataHealthCandle(t, ctx, store, symbol, start.Add(2*time.Minute))

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
		t.Fatal("invalid data health task not listed")
	}
	if found.DataHealth != data.DataSyncHealthInvalid {
		t.Fatalf("data health = %q, want invalid", found.DataHealth)
	}
	if found.GapSummary != nil {
		t.Fatalf("gap summary = %#v, want nil for contiguous invalid data", found.GapSummary)
	}
}

func insertLegacyInvalidDataHealthCandle(t *testing.T, ctx context.Context, store *Store, symbol string, openTime time.Time) {
	t.Helper()
	dropPositivePriceConstraint(t, ctx, store)
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		VALUES ('binance', $1, '1m', $2, $3, 0, 1, 0, 0.5, 1, true, now())`,
		symbol,
		openTime,
		openTime.Add(time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	ensurePositivePriceConstraint(t, ctx, store)
}
