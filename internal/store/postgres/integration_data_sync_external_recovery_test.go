package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/adapter/binance"
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
	if countIntegrationMarketCandles(t, ctx, store, symbol) != 0 {
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

func countIntegrationMarketCandles(t *testing.T, ctx context.Context, store *Store, symbol string) int {
	t.Helper()

	var count int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)::int
		  FROM market_candles
		 WHERE exchange = 'binance'
		   AND symbol = $1
		   AND interval = '1m'`,
		symbol,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
