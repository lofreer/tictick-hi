package postgres

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/datasync"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestIntegrationDataSyncRunnerResumesRealtimeTaskFromExpiredLease(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")

	id := integrationID("dst")
	symbol := integrationSymbol("RS")
	latest := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Minute)
	resumeFrom := latest.Add(-2 * time.Minute)
	wantCursor := latest.Add(2 * time.Minute)
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "crashed-worker")
	for minute := -2; minute <= 0; minute++ {
		insertIntegrationCandle(t, ctx, store, integrationResumeCandle(symbol, latest.Add(time.Duration(minute)*time.Minute), "1.2"))
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = 'binance'`)
	})
	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET last_synced_open_time = $2,
		       locked_until = now() - interval '1 second',
		       heartbeat_at = now() - interval '5 minutes',
		       created_at = '1999-01-01T00:00:00Z'::timestamptz
		 WHERE id = $1`,
		id,
		latest,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_exchange_backoffs (exchange, next_attempt_at, last_error, updated_at)
		VALUES ('binance', now() - interval '1 second', 'temporary EOF', now())`); err != nil {
		t.Fatal(err)
	}

	client := &recordingIntegrationMarketClient{
		candles: []data.Candle{
			integrationResumeCandle(symbol, latest.Add(-2*time.Minute), "1.8"),
			integrationResumeCandle(symbol, latest.Add(-1*time.Minute), "1.8"),
			integrationResumeCandle(symbol, latest, "1.8"),
			integrationResumeCandle(symbol, latest.Add(time.Minute), "1.8"),
			integrationResumeCandle(symbol, wantCursor, "1.8"),
		},
	}
	runner := datasync.NewRunner(
		store,
		exchange.NewRegistry(map[string]exchange.MarketDataClient{"binance": client}),
		datasync.Config{
			WorkerID:          "restart-worker",
			LeaseTTL:          time.Minute,
			HeartbeatInterval: time.Second,
			BatchLimit:        10,
			OverlapCandles:    2,
		},
	)

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if client.calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", client.calls)
	}
	if !client.request.From.Equal(resumeFrom) {
		t.Fatalf("resume request from = %s, want %s", client.request.From, resumeFrom)
	}
	if client.request.Symbol != symbol || client.request.Interval != "1m" || client.request.Limit != 10 {
		t.Fatalf("unexpected resume request: %#v", client.request)
	}

	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusRunning || !row.syncEnabled || !row.realtimeEnabled {
		t.Fatalf("realtime task should remain running and enabled after resume: %#v", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("saved realtime result should release the claimed lease: %#v", row)
	}
	if row.nextAttemptAt.Valid || row.lastError != "" {
		t.Fatalf("successful resume should clear retry/error state: %#v", row)
	}
	if count := countIntegrationExchangeBackoffs(t, ctx, store, "binance"); count != 0 {
		t.Fatalf("successful resume should clear expired exchange backoff, count=%d", count)
	}

	var lastSynced time.Time
	if err := store.pool.QueryRow(ctx, `
		SELECT last_synced_open_time
		  FROM data_sync_tasks
		 WHERE id = $1`,
		id,
	).Scan(&lastSynced); err != nil {
		t.Fatal(err)
	}
	if !lastSynced.Equal(wantCursor) {
		t.Fatalf("last synced cursor = %s, want %s", lastSynced, wantCursor)
	}

	var candleCount, distinctOpenCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)::int, count(DISTINCT open_time)::int
		  FROM market_candles
		 WHERE exchange = 'binance'
		   AND symbol = $1
		   AND interval = '1m'`,
		symbol,
	).Scan(&candleCount, &distinctOpenCount); err != nil {
		t.Fatal(err)
	}
	if candleCount != 5 || distinctOpenCount != 5 {
		t.Fatalf("upsert should leave five unique candles, count=%d distinct=%d", candleCount, distinctOpenCount)
	}
	candles, err := store.ListNativeCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertOpenTimes(
		t,
		candles,
		latest.Add(-2*time.Minute),
		latest.Add(-time.Minute),
		latest,
		latest.Add(time.Minute),
		wantCursor,
	)
	for _, candle := range candles {
		if normalizeDecimalText(candle.Close) != "1.8" {
			t.Fatalf("overlap candle should be updated by upsert, got %#v", candle)
		}
	}

	tasks, err := store.ListDataSyncTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var listed *data.DataSyncTask
	for index := range tasks {
		if tasks[index].ID == id {
			listed = &tasks[index]
			break
		}
	}
	if listed == nil {
		t.Fatal("resumed task should be visible in data sync task list")
	}
	if listed.LatestSyncedOpenTime == nil || !listed.LatestSyncedOpenTime.Equal(wantCursor) {
		t.Fatalf("listed task cursor = %#v, want %s", listed.LatestSyncedOpenTime, wantCursor)
	}
	if listed.DataHealth != data.DataSyncHealthSyncing {
		t.Fatalf("listed data health = %q, want syncing", listed.DataHealth)
	}
}

func TestIntegrationDataSyncRunnerDoesNotPersistOpenFetchedCandle(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")

	id := integrationID("dst_open_candle")
	symbol := integrationSymbol("OC")
	start := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Minute)
	end := start.Add(3 * time.Minute)
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusPending, true, false, "")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = 'binance'`)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})
	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET start_time = $2,
		       end_time = $3
		 WHERE id = $1`,
		id,
		start,
		end,
	); err != nil {
		t.Fatal(err)
	}

	openCandle := integrationResumeCandle(symbol, start.Add(time.Minute), "1.9")
	openCandle.IsClosed = false
	client := &recordingIntegrationMarketClient{
		candles: []data.Candle{
			integrationResumeCandle(symbol, start, "1.8"),
			openCandle,
		},
	}
	runner := datasync.NewRunner(
		store,
		exchange.NewRegistry(map[string]exchange.MarketDataClient{"binance": client}),
		datasync.Config{
			WorkerID:          "open-candle-worker",
			LeaseTTL:          time.Minute,
			HeartbeatInterval: time.Second,
			BatchLimit:        10,
		},
	)

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if client.calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", client.calls)
	}

	var candleCount, openCandleCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)::int,
		       (count(*) FILTER (WHERE is_closed = false))::int
		  FROM market_candles
		 WHERE exchange = 'binance'
		   AND symbol = $1
		   AND interval = '1m'`,
		symbol,
	).Scan(&candleCount, &openCandleCount); err != nil {
		t.Fatal(err)
	}
	if candleCount != 1 || openCandleCount != 0 {
		t.Fatalf("persisted candle count=%d open=%d, want one closed candle only", candleCount, openCandleCount)
	}

	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusPending || !row.syncEnabled || row.realtimeEnabled {
		t.Fatalf("task should remain pending for missing closed candles: %#v", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("saved result should release lease: %#v", row)
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
	if !latestSynced.Valid || !latestSynced.Time.Equal(start) {
		t.Fatalf("latest synced cursor = %#v, want %s", latestSynced, start)
	}
}

func TestIntegrationDataSyncRunnerDoesNotCompleteUnboundedOpenOnlyTask(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")

	id := integrationID("dst_open_unbounded")
	symbol := integrationSymbol("OU")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusPending, true, false, "")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = 'binance'`)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})

	openCandle := integrationResumeCandle(symbol, time.Now().UTC().Add(-time.Minute).Truncate(time.Minute), "1.9")
	openCandle.IsClosed = false
	client := &recordingIntegrationMarketClient{candles: []data.Candle{openCandle}}
	runner := datasync.NewRunner(
		store,
		exchange.NewRegistry(map[string]exchange.MarketDataClient{"binance": client}),
		datasync.Config{
			WorkerID:          "open-unbounded-worker",
			LeaseTTL:          time.Minute,
			HeartbeatInterval: time.Second,
			BatchLimit:        10,
		},
	)

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if client.calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", client.calls)
	}

	var candleCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)::int
		  FROM market_candles
		 WHERE exchange = 'binance'
		   AND symbol = $1
		   AND interval = '1m'`,
		symbol,
	).Scan(&candleCount); err != nil {
		t.Fatal(err)
	}
	if candleCount != 0 {
		t.Fatalf("open-only unbounded task should not persist candles, count=%d", candleCount)
	}

	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusPending || !row.syncEnabled || row.realtimeEnabled {
		t.Fatalf("open-only unbounded task should remain pending: %#v", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("saved result should release lease: %#v", row)
	}

	var latestSynced, finishedAt sql.NullTime
	if err := store.pool.QueryRow(ctx, `
		SELECT last_synced_open_time, finished_at
		  FROM data_sync_tasks
		 WHERE id = $1`,
		id,
	).Scan(&latestSynced, &finishedAt); err != nil {
		t.Fatal(err)
	}
	if latestSynced.Valid {
		t.Fatalf("open-only unbounded task should not advance cursor: %#v", latestSynced)
	}
	if finishedAt.Valid {
		t.Fatalf("open-only unbounded task should not finish: %#v", finishedAt)
	}
}

func normalizeDecimalText(value string) string {
	value = strings.TrimRight(value, "0")
	value = strings.TrimRight(value, ".")
	if value == "" {
		return "0"
	}
	return value
}

func TestIntegrationSaveDataSyncResultKeepsFutureExchangeBackoff(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")

	id := integrationID("dst")
	symbol := integrationSymbol("FB")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "save-worker")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
	})
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_exchange_backoffs (exchange, next_attempt_at, last_error, updated_at)
		VALUES ('binance', now() + interval '1 hour', 'temporary EOF', now())`); err != nil {
		t.Fatal(err)
	}

	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:   id,
		WorkerID: "save-worker",
	}); err != nil {
		t.Fatal(err)
	}
	if count := countIntegrationExchangeBackoffs(t, ctx, store, "binance"); count != 1 {
		t.Fatalf("successful sync result should keep future exchange backoff, count=%d", count)
	}
}

func TestIntegrationSaveDataSyncResultRejectsMismatchedCandleTarget(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst")
	symbol := integrationSymbol("MT")
	wrongSymbol := integrationSymbol("MW")
	openTime := time.Date(2026, 6, 29, 2, 30, 0, 0, time.UTC)
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, false, "save-worker")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol IN ($1, $2)`, symbol, wrongSymbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol IN ($1, $2)`, symbol, wrongSymbol)
	})

	err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:   id,
		WorkerID: "save-worker",
		Candles: []data.Candle{
			integrationResumeCandle(wrongSymbol, openTime, "1.8"),
		},
		LastOpenTime: &openTime,
		Completed:    true,
	})
	if err == nil {
		t.Fatal("expected mismatched candle target error")
	}
	if !strings.Contains(err.Error(), "target does not match") {
		t.Fatalf("error = %v, want target mismatch", err)
	}

	var candleCount int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)::int
		  FROM market_candles
		 WHERE symbol IN ($1, $2)`,
		symbol,
		wrongSymbol,
	).Scan(&candleCount); err != nil {
		t.Fatal(err)
	}
	if candleCount != 0 {
		t.Fatalf("mismatched result should not write candles, count=%d", candleCount)
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
		t.Fatalf("mismatched result should not advance cursor: %#v", latestSynced)
	}
	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusRunning || !row.syncEnabled {
		t.Fatalf("mismatched result should not transition task: %#v", row)
	}
}

type recordingIntegrationMarketClient struct {
	candles []data.Candle
	request exchange.CandleRequest
	calls   int
}

func (client *recordingIntegrationMarketClient) FetchCandles(
	_ context.Context,
	request exchange.CandleRequest,
) ([]data.Candle, error) {
	client.calls++
	client.request = request
	return client.candles, nil
}

func integrationResumeCandle(symbol string, openTime time.Time, close string) data.Candle {
	return data.Candle{
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  "1m",
		OpenTime:  openTime,
		CloseTime: openTime.Add(time.Minute),
		Open:      "1",
		High:      "2",
		Low:       "1",
		Close:     close,
		Volume:    "10",
		IsClosed:  true,
	}
}

func countIntegrationExchangeBackoffs(t *testing.T, ctx context.Context, store *Store, exchangeName string) int {
	t.Helper()

	var count int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)::int
		  FROM data_sync_exchange_backoffs
		 WHERE exchange = $1`,
		exchangeName,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
