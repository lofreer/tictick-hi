package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestIntegrationCandleProviderAggregatesAndReportsGaps(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	symbol := integrationSymbol("CP")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	start := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	for index := 0; index < 5; index++ {
		insertIntegrationCandle(t, ctx, store, data.Candle{
			Exchange:  "binance",
			Symbol:    symbol,
			Interval:  "1m",
			OpenTime:  start.Add(time.Duration(index) * time.Minute),
			CloseTime: start.Add(time.Duration(index+1) * time.Minute),
			Open:      fmt.Sprintf("%d", 100+index),
			High:      fmt.Sprintf("%d", 101+index),
			Low:       fmt.Sprintf("%d", 99+index),
			Close:     fmt.Sprintf("%d", 100+index),
			Volume:    fmt.Sprintf("%d", index+1),
			IsClosed:  true,
		})
	}
	insertIntegrationCandle(t, ctx, store, data.Candle{
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  "1m",
		OpenTime:  start.Add(10 * time.Minute),
		CloseTime: start.Add(11 * time.Minute),
		Open:      "120",
		High:      "121",
		Low:       "119",
		Close:     "120",
		Volume:    "1",
		IsClosed:  true,
	})

	aggregated, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "5m",
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if aggregated.Source != data.CandleSourceAggregated ||
		aggregated.BaseInterval != "1m" ||
		aggregated.Health != data.CandleHealthGap {
		t.Fatalf("unexpected aggregated metadata: %#v", aggregated)
	}
	if len(aggregated.Candles) != 2 ||
		aggregated.Candles[0].Interval != "5m" ||
		aggregated.Candles[0].Volume != "15" {
		t.Fatalf("unexpected aggregated candles: %#v", aggregated.Candles)
	}
	if !aggregated.Candles[0].IsClosed || aggregated.Candles[1].IsClosed {
		t.Fatalf("unexpected aggregated candle close state: %#v", aggregated.Candles)
	}
	if len(aggregated.Gaps) != 1 || aggregated.Gaps[0].MissingCandles != 5 {
		t.Fatalf("unexpected base gaps: %#v", aggregated.Gaps)
	}

	native, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if native.Source != data.CandleSourceNative ||
		native.Health != data.CandleHealthGap ||
		len(native.Gaps) != 1 ||
		native.Gaps[0].MissingCandles != 5 {
		t.Fatalf("unexpected native gap result: %#v", native)
	}
}

func TestIntegrationListNativeCandlesUsesLatestWindowWithoutRange(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	symbol := integrationSymbol("LW")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	start := time.Date(2026, 6, 27, 1, 0, 0, 0, time.UTC)
	for index := 0; index < 10; index++ {
		insertIntegrationCandle(t, ctx, store, data.Candle{
			Exchange:  "binance",
			Symbol:    symbol,
			Interval:  "1m",
			OpenTime:  start.Add(time.Duration(index) * time.Minute),
			CloseTime: start.Add(time.Duration(index+1) * time.Minute),
			Open:      fmt.Sprintf("%d", index),
			High:      fmt.Sprintf("%d", index+1),
			Low:       fmt.Sprintf("%d", index),
			Close:     fmt.Sprintf("%d", index),
			Volume:    "1",
			IsClosed:  true,
		})
	}

	latest, err := store.ListNativeCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertOpenTimes(t, latest, start.Add(7*time.Minute), start.Add(8*time.Minute), start.Add(9*time.Minute))

	from := start.Add(2 * time.Minute)
	inRange, err := store.ListNativeCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		From:     &from,
		Limit:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertOpenTimes(t, inRange, start.Add(2*time.Minute), start.Add(3*time.Minute), start.Add(4*time.Minute))

	aggregated, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "5m",
		Limit:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(aggregated.Candles) != 1 ||
		!aggregated.Candles[0].OpenTime.Equal(start.Add(5*time.Minute)) ||
		aggregated.Candles[0].Volume != "5" {
		t.Fatalf("expected latest 5m aggregation, got %#v", aggregated.Candles)
	}
}

func TestIntegrationListNativeCandlesClampsOversizedLimit(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	symbol := integrationSymbol("LC")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})

	start := time.Date(2026, 6, 27, 2, 0, 0, 0, time.UTC)
	count := data.DefaultCandleLimit + 5
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		SELECT 'binance',
		       $1,
		       '1m',
		       $2::timestamptz + (series.idx * interval '1 minute'),
		       $2::timestamptz + ((series.idx + 1) * interval '1 minute'),
		       series.idx::numeric,
		       (series.idx + 1)::numeric,
		       series.idx::numeric,
		       series.idx::numeric,
		       1,
		       true,
		       now()
		  FROM generate_series(0, $3::int - 1) AS series(idx)`,
		symbol,
		start,
		count,
	); err != nil {
		t.Fatal(err)
	}

	candles, err := store.ListNativeCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    data.MaxCandleLimit + 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != count {
		t.Fatalf("candles len = %d, want %d", len(candles), count)
	}
	assertOpenTimes(t, candles[:1], start)
	assertOpenTimes(t, candles[len(candles)-1:], start.Add(time.Duration(count-1)*time.Minute))
}

func TestIntegrationDataSyncRetryReleasesAndReclaimsTask(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst")
	symbol := integrationSymbol("RT")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "retry-worker")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
	})

	if err := store.RecordDataSyncRetry(
		ctx,
		id,
		exchange.NewTemporaryError("binance klines temporary unavailable: api.binance.com: Get EOF", nil),
	); err != nil {
		t.Fatal(err)
	}

	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusRunning || !row.syncEnabled || !row.realtimeEnabled {
		t.Fatalf("retry should keep realtime task claimable: %#v", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("retry should release lease: %#v", row)
	}
	if !strings.Contains(row.lastError, "temporary unavailable") || strings.Contains(row.lastError, "\n") {
		t.Fatalf("retry should store normalized error: %q", row.lastError)
	}

	claimed, ok, err := store.ClaimDataSyncTask(ctx, "retry-worker-2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || claimed.ID != id {
		t.Fatalf("expected retried task to be reclaimed, ok=%v task=%#v", ok, claimed)
	}
	row = readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusRunning || !row.lockedBy.Valid || row.lockedBy.String != "retry-worker-2" {
		t.Fatalf("unexpected reclaimed row: %#v", row)
	}
}

func TestIntegrationDataSyncPermanentFailureStopsTask(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst")
	symbol := integrationSymbol("PF")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "failed-worker")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
	})

	if err := store.MarkDataSyncFailed(ctx, id, errors.New("invalid symbol")); err != nil {
		t.Fatal(err)
	}

	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusFailed || row.syncEnabled || row.realtimeEnabled {
		t.Fatalf("permanent failure should stop task: %#v", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("permanent failure should release lease: %#v", row)
	}
	if row.lastError != "invalid symbol" {
		t.Fatalf("unexpected last error: %q", row.lastError)
	}

	claimable := false
	if err := store.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			  FROM data_sync_tasks
			 WHERE id = $1
			   AND (sync_enabled = true OR realtime_enabled = true)
			   AND status IN ($2, $3)
			   AND (locked_until IS NULL OR locked_until < now())
		)`,
		id,
		data.TaskStatusPending,
		data.TaskStatusRunning,
	).Scan(&claimable); err != nil {
		t.Fatal(err)
	}
	if claimable {
		t.Fatal("permanently failed task should not remain claimable")
	}
}

func TestIntegrationRetryDataSyncTaskRestoresFailedTask(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst")
	symbol := integrationSymbol("RF")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusFailed, false, false, "failed-worker")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
	})

	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET last_error = 'invalid symbol'
		 WHERE id = $1`,
		id,
	); err != nil {
		t.Fatal(err)
	}

	retried, err := store.RetryDataSyncTask(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Status != data.TaskStatusPending || !retried.SyncEnabled || retried.RealtimeEnabled {
		t.Fatalf("unexpected retried task: %#v", retried)
	}
	if retried.LastError != "" {
		t.Fatalf("retry should clear last error: %#v", retried)
	}
	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusPending || !row.syncEnabled || row.realtimeEnabled {
		t.Fatalf("retry should restore a claimable one-shot sync task: %#v", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("retry should clear failed lease fields: %#v", row)
	}
	if row.lastError != "" {
		t.Fatalf("retry should clear stored error: %q", row.lastError)
	}

	claimable := false
	if err := store.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			  FROM data_sync_tasks
			 WHERE id = $1
			   AND (sync_enabled = true OR realtime_enabled = true)
			   AND status IN ($2, $3)
			   AND (locked_until IS NULL OR locked_until < now())
		)`,
		id,
		data.TaskStatusPending,
		data.TaskStatusRunning,
	).Scan(&claimable); err != nil {
		t.Fatal(err)
	}
	if !claimable {
		t.Fatal("expected retried task to satisfy claim predicate")
	}
}

func TestIntegrationRetryDataSyncTaskRejectsRunningTask(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	id := integrationID("dst")
	symbol := integrationSymbol("RR")
	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusRunning, true, true, "active-worker")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
	})

	_, err := store.RetryDataSyncTask(ctx, id)
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("retry running task error = %v, want invalid state", err)
	}
	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusRunning || !row.lockedBy.Valid || row.lockedBy.String != "active-worker" {
		t.Fatalf("retry should not mutate running task lease: %#v", row)
	}
}

func openIntegrationStore(t *testing.T) *Store {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TICTICK_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set TICTICK_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := testContext(t)
	defer cancel()

	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)

	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return store
}

func testContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 15*time.Second)
}

func integrationID(prefix string) string {
	return fmt.Sprintf("%s_it_%d", prefix, time.Now().UTC().UnixNano())
}

func integrationSymbol(prefix string) string {
	return fmt.Sprintf("IT%s%dUSDT", prefix, time.Now().UTC().UnixNano())
}

func insertIntegrationCandle(t *testing.T, ctx context.Context, store *Store, candle data.Candle) {
	t.Helper()

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7::numeric, $8::numeric,
		        $9::numeric, $10::numeric, $11, now())`,
		candle.Exchange,
		candle.Symbol,
		candle.Interval,
		candle.OpenTime,
		candle.CloseTime,
		candle.Open,
		candle.High,
		candle.Low,
		candle.Close,
		candle.Volume,
		candle.IsClosed,
	); err != nil {
		t.Fatal(err)
	}
}

func insertIntegrationSyncTask(
	t *testing.T,
	ctx context.Context,
	store *Store,
	id string,
	symbol string,
	status data.TaskStatus,
	syncEnabled bool,
	realtimeEnabled bool,
	lockedBy string,
) {
	t.Helper()

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_tasks (
			id, exchange, symbol, interval, sync_enabled, realtime_enabled, status,
			locked_by, locked_until, heartbeat_at, started_at, created_at, updated_at
		)
		VALUES ($1, 'binance', $2, '1m', $3, $4, $5, $6, now() + interval '1 minute',
		        now(), now(), '2000-01-01T00:00:00Z'::timestamptz, now())`,
		id,
		symbol,
		syncEnabled,
		realtimeEnabled,
		status,
		lockedBy,
	); err != nil {
		t.Fatal(err)
	}
}

func assertOpenTimes(t *testing.T, candles []data.Candle, expected ...time.Time) {
	t.Helper()

	if len(candles) != len(expected) {
		t.Fatalf("candles = %d, want %d: %#v", len(candles), len(expected), candles)
	}
	for index, expectedTime := range expected {
		if !candles[index].OpenTime.Equal(expectedTime) {
			t.Fatalf("candle %d open time = %s, want %s; candles=%#v", index, candles[index].OpenTime, expectedTime, candles)
		}
	}
}

type integrationSyncTaskRow struct {
	status          data.TaskStatus
	syncEnabled     bool
	realtimeEnabled bool
	lockedBy        sql.NullString
	lockedUntil     sql.NullTime
	heartbeatAt     sql.NullTime
	lastError       string
}

func readIntegrationSyncTask(
	t *testing.T,
	ctx context.Context,
	store *Store,
	id string,
) integrationSyncTaskRow {
	t.Helper()

	var row integrationSyncTaskRow
	if err := store.pool.QueryRow(ctx, `
		SELECT status, sync_enabled, realtime_enabled, locked_by, locked_until,
		       heartbeat_at, COALESCE(last_error, '')
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
		&row.lastError,
	); err != nil {
		t.Fatal(err)
	}
	return row
}
