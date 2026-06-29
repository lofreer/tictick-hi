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
		ptrTime(start.Add(4*time.Minute)),
		nil,
		"",
	)
	for _, minute := range []int{0, 1} {
		insertIntegrationCandle(t, ctx, store, integrationDataHealthCandle(symbol, start, minute))
	}
	insertLegacyInvalidDataHealthCandle(t, ctx, store, symbol, start.Add(2*time.Minute))
	insertLegacyInvalidDataHealthCandle(t, ctx, store, symbol, start.Add(3*time.Minute))
	insertLegacyInvalidCloseDataHealthCandle(t, ctx, store, symbol, start.Add(4*time.Minute))

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
	if found.InvalidSummary == nil {
		t.Fatal("invalid task should expose invalid summary")
	}
	if found.InvalidSummary.Count != 3 {
		t.Fatalf("invalid summary count = %d, want 3", found.InvalidSummary.Count)
	}
	if found.InvalidSummary.FirstIssue == nil {
		t.Fatal("invalid task should expose first invalid issue")
	}
	if found.InvalidSummary.FirstIssue.Code != "invalid_open_price" ||
		found.InvalidSummary.FirstIssue.Message != "open price value must be positive" ||
		found.InvalidSummary.FirstIssue.OpenTime == nil ||
		!found.InvalidSummary.FirstIssue.OpenTime.Equal(start.Add(2*time.Minute)) {
		t.Fatalf("unexpected invalid issue: %#v", found.InvalidSummary.FirstIssue)
	}
	if found.GapSummary != nil {
		t.Fatalf("gap summary = %#v, want nil for contiguous invalid data", found.GapSummary)
	}

	issues, err := store.ListDataSyncTaskInvalidIssues(ctx, taskID, data.DataSyncInvalidIssueQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if issues.TaskID != taskID ||
		issues.Limited ||
		issues.TotalCount != 2 ||
		issues.ReturnedCount != 2 ||
		issues.IssueLimit != maxDataSyncInvalidIssues ||
		issues.Offset != 0 ||
		len(issues.Issues) != 2 {
		t.Fatalf("unexpected invalid issue list: %#v", issues)
	}
	issue := issues.Issues[0]
	if issue.Code != "invalid_open_price" ||
		issue.Message != "open price value must be positive" ||
		issue.OpenTime == nil ||
		!issue.OpenTime.Equal(start.Add(2*time.Minute)) {
		t.Fatalf("unexpected invalid issue detail: %#v", issue)
	}

	secondPage, err := store.ListDataSyncTaskInvalidIssues(ctx, taskID, data.DataSyncInvalidIssueQuery{
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if secondPage.TaskID != taskID ||
		secondPage.Limited ||
		secondPage.TotalCount != 2 ||
		secondPage.ReturnedCount != 1 ||
		secondPage.IssueLimit != 1 ||
		secondPage.Offset != 1 ||
		len(secondPage.Issues) != 1 ||
		secondPage.Issues[0].OpenTime == nil ||
		!secondPage.Issues[0].OpenTime.Equal(start.Add(3*time.Minute)) {
		t.Fatalf("unexpected paged invalid issue list: %#v", secondPage)
	}

	filtered, err := store.ListDataSyncTaskInvalidIssues(ctx, taskID, data.DataSyncInvalidIssueQuery{
		Code: "invalid_close_price",
		From: ptrTime(start.Add(4 * time.Minute)),
		To:   ptrTime(start.Add(4 * time.Minute)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.TaskID != taskID ||
		filtered.Limited ||
		filtered.TotalCount != 1 ||
		filtered.ReturnedCount != 1 ||
		filtered.Offset != 0 ||
		len(filtered.Issues) != 1 ||
		filtered.Issues[0].Code != "invalid_close_price" ||
		filtered.Issues[0].OpenTime == nil ||
		!filtered.Issues[0].OpenTime.Equal(start.Add(4*time.Minute)) {
		t.Fatalf("unexpected filtered invalid issue list: %#v", filtered)
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

func insertLegacyInvalidCloseDataHealthCandle(t *testing.T, ctx context.Context, store *Store, symbol string, openTime time.Time) {
	t.Helper()
	dropPositivePriceConstraint(t, ctx, store)
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		VALUES ('binance', $1, '1m', $2, $3, 1, 1.5, 0.5, 0, 1, true, now())`,
		symbol,
		openTime,
		openTime.Add(time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	ensurePositivePriceConstraint(t, ctx, store)
}
