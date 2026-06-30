package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/adapter/binance"
	"github.com/lofreer/tictick-hi/internal/adapter/okx"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/datasync"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestIntegrationDataSyncRunnerRecoversAfterBinanceRetryAfter(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")

	id := integrationID("dst")
	symbol := integrationSymbol("BR")
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(4 * time.Minute)
	retryAfter := 2 * time.Minute
	market := newRetryAfterBinanceServer(symbol, start, retryAfter)
	defer market.Close()

	insertIntegrationSyncTask(t, ctx, store, id, symbol, data.TaskStatusPending, true, false, "")
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
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = 'binance'`)
	})

	runner := datasync.NewRunner(
		store,
		exchange.NewRegistry(map[string]exchange.MarketDataClient{
			"binance": binance.NewMarketClientForURL(market.URL, market.Client()),
		}),
		datasync.Config{
			WorkerID:          "retry-after-worker",
			LeaseTTL:          time.Minute,
			HeartbeatInterval: time.Second,
			BatchLimit:        10,
			FetchRetries:      3,
			RetryBackoff:      30 * time.Second,
			MaxRetryBackoff:   5 * time.Minute,
		},
	)
	now := time.Now().UTC()

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if hits := market.Hits(); hits != 1 {
		t.Fatalf("binance hits after retry-after run = %d, want 1", hits)
	}

	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusPending || !row.syncEnabled || row.realtimeEnabled {
		t.Fatalf("retry-after task state = %#v, want pending sync task", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("retry-after task lease should be released: %#v", row)
	}
	if row.nextAttemptAt.Valid == false {
		t.Fatalf("retry-after task should have next attempt: %#v", row)
	}
	if row.nextAttemptAt.Time.Before(now.Add(retryAfter - 5*time.Second)) {
		t.Fatalf("next attempt = %s, want at least Retry-After %s", row.nextAttemptAt.Time, retryAfter)
	}
	if row.lastError == "" {
		t.Fatal("retry-after task should store a sanitized error")
	}
	assertIntegrationNoRequestURLLeak(t, row.lastError)
	if countIntegrationMarketCandles(t, ctx, store, "binance", symbol) != 0 {
		t.Fatal("retry-after failure should not persist candles")
	}

	backoffUntil := readIntegrationExchangeBackoff(t, ctx, store, "binance")
	if !backoffUntil.Valid {
		t.Fatal("retry-after should create exchange backoff")
	}
	if backoffUntil.Time.Before(now.Add(retryAfter - 5*time.Second)) {
		t.Fatalf("exchange backoff = %s, want at least Retry-After %s", backoffUntil.Time, retryAfter)
	}

	forceIntegrationDataSyncRetryDue(t, ctx, store, id, "binance")
	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if hits := market.Hits(); hits != 2 {
		t.Fatalf("binance hits after recovery run = %d, want 2", hits)
	}

	row = readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusSucceeded || row.syncEnabled || row.realtimeEnabled {
		t.Fatalf("recovered task state = %#v, want succeeded one-shot sync", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("recovered task lease should be released: %#v", row)
	}
	if row.nextAttemptAt.Valid || row.lastError != "" {
		t.Fatalf("recovered task should clear retry state: %#v", row)
	}
	if count := countIntegrationExchangeBackoffs(t, ctx, store, "binance"); count != 0 {
		t.Fatalf("successful recovery should clear exchange backoff, count=%d", count)
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
	assertOpenTimes(t, candles, start, start.Add(time.Minute), start.Add(2*time.Minute), start.Add(3*time.Minute), start.Add(4*time.Minute))

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
		t.Fatal("recovered task should be visible in data sync task list")
	}
	if listed.LatestSyncedOpenTime == nil || !listed.LatestSyncedOpenTime.Equal(start.Add(4*time.Minute)) {
		t.Fatalf("listed cursor = %#v, want %s", listed.LatestSyncedOpenTime, start.Add(4*time.Minute))
	}
	if listed.DataHealth != data.DataSyncHealthOK {
		t.Fatalf("listed data health = %q, want ok", listed.DataHealth)
	}
}

func TestIntegrationDataSyncRunnerRecoversAfterOKXRateLimitCode(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "okx")

	id := integrationID("dst_okx")
	symbol := strings.TrimSuffix(integrationSymbol("OKX"), "USDT") + "-USDT"
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(4 * time.Minute)
	market := newRateLimitOKXServer(symbol, start)
	defer market.Close()

	insertIntegrationSyncTaskForExchange(t, ctx, store, "okx", id, symbol, data.TaskStatusPending, true, false, "")
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
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE exchange = 'okx' AND symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE exchange = 'okx' AND symbol = $1`, symbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange = 'okx'`)
	})

	runner := datasync.NewRunner(
		store,
		exchange.NewRegistry(map[string]exchange.MarketDataClient{
			"okx": okx.NewMarketClientForURL(market.URL, market.Client()),
		}),
		datasync.Config{
			WorkerID:          "okx-rate-limit-worker",
			LeaseTTL:          time.Minute,
			HeartbeatInterval: time.Second,
			BatchLimit:        10,
			FetchRetries:      1,
			RetryDelay:        time.Nanosecond,
			RetryBackoff:      30 * time.Second,
			MaxRetryBackoff:   5 * time.Minute,
		},
	)
	now := time.Now().UTC()

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if hits := market.Hits(); hits != 2 {
		t.Fatalf("okx hits after exhausted retry run = %d, want 2", hits)
	}

	row := readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusPending || !row.syncEnabled || row.realtimeEnabled {
		t.Fatalf("okx retry task state = %#v, want pending sync task", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("okx retry task lease should be released: %#v", row)
	}
	if row.nextAttemptAt.Valid == false {
		t.Fatalf("okx retry task should have next attempt: %#v", row)
	}
	if row.nextAttemptAt.Time.Before(now.Add(25 * time.Second)) {
		t.Fatalf("okx next attempt = %s, want retry backoff", row.nextAttemptAt.Time)
	}
	if row.lastError == "" {
		t.Fatal("okx retry task should store a sanitized error")
	}
	assertIntegrationNoRequestURLLeak(t, row.lastError)
	if countIntegrationMarketCandles(t, ctx, store, "okx", symbol) != 0 {
		t.Fatal("okx temporary failure should not persist candles")
	}

	backoffUntil := readIntegrationExchangeBackoff(t, ctx, store, "okx")
	if !backoffUntil.Valid {
		t.Fatal("okx temporary failure should create exchange backoff")
	}
	if backoffUntil.Time.Before(now.Add(25 * time.Second)) {
		t.Fatalf("okx exchange backoff = %s, want retry backoff", backoffUntil.Time)
	}

	forceIntegrationDataSyncRetryDue(t, ctx, store, id, "okx")
	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if hits := market.Hits(); hits != 3 {
		t.Fatalf("okx hits after recovery run = %d, want 3", hits)
	}

	row = readIntegrationSyncTask(t, ctx, store, id)
	if row.status != data.TaskStatusSucceeded || row.syncEnabled || row.realtimeEnabled {
		t.Fatalf("okx recovered task state = %#v, want succeeded one-shot sync", row)
	}
	if row.lockedBy.Valid || row.lockedUntil.Valid || row.heartbeatAt.Valid {
		t.Fatalf("okx recovered task lease should be released: %#v", row)
	}
	if row.nextAttemptAt.Valid || row.lastError != "" {
		t.Fatalf("okx recovered task should clear retry state: %#v", row)
	}
	if count := countIntegrationExchangeBackoffs(t, ctx, store, "okx"); count != 0 {
		t.Fatalf("okx successful recovery should clear exchange backoff, count=%d", count)
	}

	candles, err := store.ListNativeCandles(ctx, data.CandleQuery{
		Exchange: "okx",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertOpenTimes(t, candles, start, start.Add(time.Minute), start.Add(2*time.Minute), start.Add(3*time.Minute), start.Add(4*time.Minute))

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
		t.Fatal("okx recovered task should be visible in data sync task list")
	}
	if listed.LatestSyncedOpenTime == nil || !listed.LatestSyncedOpenTime.Equal(start.Add(4*time.Minute)) {
		t.Fatalf("okx listed cursor = %#v, want %s", listed.LatestSyncedOpenTime, start.Add(4*time.Minute))
	}
	if listed.DataHealth != data.DataSyncHealthOK {
		t.Fatalf("okx listed data health = %q, want ok", listed.DataHealth)
	}
}

type retryAfterBinanceServer struct {
	*httptest.Server

	mu         sync.Mutex
	hits       int
	symbol     string
	start      time.Time
	retryAfter time.Duration
}

func newRetryAfterBinanceServer(symbol string, start time.Time, retryAfter time.Duration) *retryAfterBinanceServer {
	server := &retryAfterBinanceServer{
		symbol:     symbol,
		start:      start,
		retryAfter: retryAfter,
	}
	server.Server = httptest.NewServer(http.HandlerFunc(server.handle))
	return server
}

func (server *retryAfterBinanceServer) Hits() int {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.hits
}

func (server *retryAfterBinanceServer) handle(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/api/v3/klines" {
		http.NotFound(response, request)
		return
	}

	server.mu.Lock()
	server.hits++
	hit := server.hits
	server.mu.Unlock()

	if hit == 1 {
		response.Header().Set("Retry-After", fmt.Sprintf("%.0f", server.retryAfter.Seconds()))
		http.Error(response, "rate limited", http.StatusTooManyRequests)
		return
	}
	if request.URL.Query().Get("symbol") != server.symbol || request.URL.Query().Get("interval") != "1m" {
		http.Error(response, "invalid request", http.StatusBadRequest)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(response).Encode(server.klines()); err != nil {
		panic(err)
	}
}

func (server *retryAfterBinanceServer) klines() [][]any {
	rows := make([][]any, 0, 5)
	for index := 0; index < 5; index++ {
		openTime := server.start.Add(time.Duration(index) * time.Minute)
		price := 100 + index
		rows = append(rows, []any{
			openTime.UnixMilli(),
			fmt.Sprintf("%d.00", price),
			fmt.Sprintf("%d.00", price+1),
			fmt.Sprintf("%d.00", price-1),
			fmt.Sprintf("%d.00", price),
			fmt.Sprintf("%d.00", 10+index),
			openTime.Add(time.Minute).Add(-time.Millisecond).UnixMilli(),
			"0",
			0,
			"0",
			"0",
			"0",
		})
	}
	return rows
}

type rateLimitOKXServer struct {
	*httptest.Server

	mu     sync.Mutex
	hits   int
	symbol string
	start  time.Time
}

func newRateLimitOKXServer(symbol string, start time.Time) *rateLimitOKXServer {
	server := &rateLimitOKXServer{
		symbol: symbol,
		start:  start,
	}
	server.Server = httptest.NewServer(http.HandlerFunc(server.handle))
	return server
}

func (server *rateLimitOKXServer) Hits() int {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.hits
}

func (server *rateLimitOKXServer) handle(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/api/v5/market/history-candles" {
		http.NotFound(response, request)
		return
	}
	if request.URL.Query().Get("instId") != server.symbol || request.URL.Query().Get("bar") != "1m" {
		http.Error(response, "invalid request", http.StatusBadRequest)
		return
	}

	server.mu.Lock()
	server.hits++
	hit := server.hits
	server.mu.Unlock()

	response.Header().Set("Content-Type", "application/json")
	if hit <= 2 {
		if err := json.NewEncoder(response).Encode(map[string]any{
			"code": "50011",
			"msg":  "Requests too frequent",
			"data": []any{},
		}); err != nil {
			panic(err)
		}
		return
	}
	if err := json.NewEncoder(response).Encode(map[string]any{
		"code": "0",
		"msg":  "",
		"data": server.candles(),
	}); err != nil {
		panic(err)
	}
}

func (server *rateLimitOKXServer) candles() [][]string {
	rows := make([][]string, 0, 5)
	for index := 4; index >= 0; index-- {
		openTime := server.start.Add(time.Duration(index) * time.Minute)
		price := 100 + index
		rows = append(rows, []string{
			fmt.Sprintf("%d", openTime.UnixMilli()),
			fmt.Sprintf("%d.00", price),
			fmt.Sprintf("%d.00", price+1),
			fmt.Sprintf("%d.00", price-1),
			fmt.Sprintf("%d.00", price),
			fmt.Sprintf("%d.00", 10+index),
			"0",
			"0",
			"1",
		})
	}
	return rows
}

func forceIntegrationDataSyncRetryDue(
	t *testing.T,
	ctx context.Context,
	store *Store,
	taskID string,
	exchangeName string,
) {
	t.Helper()

	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET next_attempt_at = now() - INTERVAL '1 second'
		 WHERE id = $1`,
		taskID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_exchange_backoffs
		   SET next_attempt_at = now() - INTERVAL '1 second'
		 WHERE exchange = $1`,
		exchangeName,
	); err != nil {
		t.Fatal(err)
	}
}

func insertIntegrationSyncTaskForExchange(
	t *testing.T,
	ctx context.Context,
	store *Store,
	exchangeName string,
	id string,
	symbol string,
	status data.TaskStatus,
	syncEnabled bool,
	realtimeEnabled bool,
	lockedBy string,
) {
	t.Helper()
	upsertIntegrationMarketInstrument(t, ctx, store, exchangeName, symbol, "active")

	var (
		leaseLockedBy  any
		leaseUntil     any
		leaseHeartbeat any
		leaseStartedAt any
		finishedAt     any
	)
	if status == data.TaskStatusRunning && lockedBy != "" {
		now := time.Now().UTC()
		leaseLockedBy = lockedBy
		leaseUntil = now.Add(time.Minute)
		leaseHeartbeat = now
		leaseStartedAt = now
	}
	if status == data.TaskStatusSucceeded || status == data.TaskStatusFailed || status == data.TaskStatusCancelled {
		finishedAt = time.Now().UTC()
	}

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_tasks (
			id, exchange, symbol, interval, sync_enabled, realtime_enabled, status,
			locked_by, locked_until, heartbeat_at, started_at, finished_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, '1m', $4, $5, $6, $7, $8, $9, $10,
		        $11, '2000-01-01T00:00:00Z'::timestamptz, now())`,
		id,
		exchangeName,
		symbol,
		syncEnabled,
		realtimeEnabled,
		status,
		leaseLockedBy,
		leaseUntil,
		leaseHeartbeat,
		leaseStartedAt,
		finishedAt,
	); err != nil {
		t.Fatal(err)
	}
}

func countIntegrationMarketCandles(t *testing.T, ctx context.Context, store *Store, exchangeName string, symbol string) int {
	t.Helper()

	var count int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)::int
		  FROM market_candles
		 WHERE exchange = $1
		   AND symbol = $2
		   AND interval = '1m'`,
		exchangeName,
		symbol,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
