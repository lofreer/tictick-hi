package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/store/postgres"
)

func TestIntegrationMarketCandleInvalidIssueRepairRouteConvergesPostgresScan(t *testing.T) {
	store, pool, ctx := openAPIIntegrationStore(t)
	server := NewServer(store, "")

	symbol := apiIntegrationSymbol("APIIC")
	username := fmt.Sprintf("api-invalid-%d", time.Now().UTC().UnixNano())
	password := "secret123"
	start := time.Date(2026, 6, 27, 10, 15, 0, 0, time.UTC)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		cleanupAPIIntegrationMarket(t, cleanupCtx, pool, symbol, username)
	})

	if _, _, err := store.EnsureOperator(ctx, data.CreateOperator{
		Username: username,
		Password: password,
		Enabled:  true,
	}); err != nil {
		t.Fatal(err)
	}
	auth := loginIntegrationOperator(t, server, username, password)

	upsertAPIIntegrationMarketInstrument(t, ctx, pool, symbol)
	insertAPIIntegrationCandle(t, ctx, pool, apiIntegrationCandle(symbol, start, 0))
	insertAPIIntegrationInvalidCandle(t, ctx, pool, symbol, start.Add(time.Minute))
	insertAPIIntegrationCandle(t, ctx, pool, apiIntegrationCandle(symbol, start, 2))

	scanPath := "/api/market/candle-invalid-issues?exchange=binance&symbol=" +
		url.QueryEscape(symbol) + "&interval=1m"
	beforeRecorder := serveAuthenticated(server, auth, http.MethodGet, scanPath, "")
	if beforeRecorder.Code != http.StatusOK {
		t.Fatalf("scan before status = %d body = %s", beforeRecorder.Code, beforeRecorder.Body.String())
	}
	var before data.MarketCandleInvalidIssueScan
	if err := json.NewDecoder(beforeRecorder.Body).Decode(&before); err != nil {
		t.Fatal(err)
	}
	if before.Window.Count != 3 || before.TotalCount != 1 || before.ReturnedCount != 1 ||
		len(before.Issues) != 1 || before.Issues[0].OpenTime == nil ||
		!before.Issues[0].OpenTime.Equal(start.Add(time.Minute)) {
		t.Fatalf("invalid scan before repair = %#v, want one persisted invalid issue", before)
	}

	body := fmt.Sprintf(
		`{"exchange":"binance","symbol":%q,"interval":"1m","openTimes":[%q]}`,
		strings.ToLower(symbol),
		start.Add(time.Minute).Format(time.RFC3339),
	)
	repairRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/candle-invalid-issues/repair", body)
	if repairRecorder.Code != http.StatusOK {
		t.Fatalf("repair status = %d body = %s", repairRecorder.Code, repairRecorder.Body.String())
	}
	var repair data.DataSyncGapRepairResult
	if err := json.NewDecoder(repairRecorder.Body).Decode(&repair); err != nil {
		t.Fatal(err)
	}
	if repair.TotalCount != 1 || repair.SkippedExisting != 0 || len(repair.CreatedTasks) != 1 ||
		repair.CreatedTasks[0].StartTime == nil ||
		!repair.CreatedTasks[0].StartTime.Equal(start.Add(time.Minute)) ||
		repair.CreatedTasks[0].EndTime == nil ||
		!repair.CreatedTasks[0].EndTime.Equal(start.Add(2*time.Minute)) {
		t.Fatalf("unexpected invalid repair result: %#v", repair)
	}
	repairTask := repair.CreatedTasks[0]

	if _, err := pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET status = $2,
		       locked_by = 'api-invalid-repair-worker',
		       locked_until = now() + interval '1 minute',
		       heartbeat_at = now()
		 WHERE id = $1`,
		repairTask.ID,
		data.TaskStatusRunning,
	); err != nil {
		t.Fatal(err)
	}

	repairedCandle := apiIntegrationCandle(symbol, start, 1)
	lastOpenTime := repairedCandle.OpenTime
	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:       repairTask.ID,
		WorkerID:     "api-invalid-repair-worker",
		Candles:      []data.Candle{repairedCandle},
		LastOpenTime: &lastOpenTime,
		Completed:    true,
	}); err != nil {
		t.Fatal(err)
	}

	afterRecorder := serveAuthenticated(server, auth, http.MethodGet, scanPath, "")
	if afterRecorder.Code != http.StatusOK {
		t.Fatalf("scan after status = %d body = %s", afterRecorder.Code, afterRecorder.Body.String())
	}
	var after data.MarketCandleInvalidIssueScan
	if err := json.NewDecoder(afterRecorder.Body).Decode(&after); err != nil {
		t.Fatal(err)
	}
	if after.Window.Count != 3 || after.TotalCount != 0 || after.ReturnedCount != 0 ||
		len(after.Issues) != 0 || after.Limited {
		t.Fatalf("invalid scan after repair result = %#v, want healthy history", after)
	}
}

func openAPIIntegrationStore(t *testing.T) (*postgres.Store, *pgxpool.Pool, context.Context) {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TICTICK_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("TICTICK_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)

	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return store, pool, ctx
}

func loginIntegrationOperator(t *testing.T, server http.Handler, username string, password string) *authTestSession {
	t.Helper()

	body := bytes.NewBufferString(fmt.Sprintf(`{"username":%q,"password":%q}`, username, password))
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	auth := &authTestSession{}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			auth.session = cookie
		}
		if cookie.Name == csrfCookieName {
			auth.csrf = cookie
		}
	}
	if auth.session == nil {
		t.Fatal("login did not set session cookie")
	}
	if auth.csrf == nil {
		t.Fatal("login did not set csrf cookie")
	}
	return auth
}

func apiIntegrationSymbol(prefix string) string {
	suffix := time.Now().UTC().UnixNano() % 1_000_000_000_000
	return fmt.Sprintf("%s%dUSDT", prefix, suffix)
}

func upsertAPIIntegrationMarketInstrument(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	symbol string,
) {
	t.Helper()

	baseAsset := strings.TrimSuffix(symbol, "USDT")
	if _, err := pool.Exec(ctx, `
		INSERT INTO market_instruments (
			exchange, symbol, base_asset, quote_asset, instrument_type, status, exchange_status, search_priority, synced_at
		)
		VALUES ('binance', $1, $2, 'USDT', 'spot', 'active', 'TRADING', 0, now())
		ON CONFLICT (exchange, symbol)
		DO UPDATE SET status = EXCLUDED.status,
		              exchange_status = EXCLUDED.exchange_status,
		              synced_at = EXCLUDED.synced_at,
		              updated_at = now()`,
		symbol,
		baseAsset,
	); err != nil {
		t.Fatal(err)
	}
}

func apiIntegrationCandle(symbol string, start time.Time, minute int) data.Candle {
	openTime := start.Add(time.Duration(minute) * time.Minute)
	price := 100 + minute
	return data.Candle{
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  "1m",
		OpenTime:  openTime,
		CloseTime: openTime.Add(time.Minute),
		Open:      fmt.Sprintf("%d", price),
		High:      fmt.Sprintf("%d", price+1),
		Low:       fmt.Sprintf("%d", price-1),
		Close:     fmt.Sprintf("%d", price),
		Volume:    "1",
		IsClosed:  true,
	}
}

func insertAPIIntegrationCandle(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	candle data.Candle,
) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7::numeric, $8::numeric, $9::numeric, $10::numeric, $11, now())
		ON CONFLICT (exchange, symbol, interval, open_time)
		DO UPDATE SET close_time = EXCLUDED.close_time,
		              open = EXCLUDED.open,
		              high = EXCLUDED.high,
		              low = EXCLUDED.low,
		              close = EXCLUDED.close,
		              volume = EXCLUDED.volume,
		              is_closed = EXCLUDED.is_closed,
		              updated_at = now()`,
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

func insertAPIIntegrationInvalidCandle(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	symbol string,
	openTime time.Time,
) {
	t.Helper()

	dropAPIPositivePriceConstraint(t, ctx, pool)
	if _, err := pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		VALUES ('binance', $1, '1m', $2, $3, 0, 101, 0, 100, 1, true, now())
		ON CONFLICT (exchange, symbol, interval, open_time)
		DO UPDATE SET close_time = EXCLUDED.close_time,
		              open = EXCLUDED.open,
		              high = EXCLUDED.high,
		              low = EXCLUDED.low,
		              close = EXCLUDED.close,
		              volume = EXCLUDED.volume,
		              is_closed = EXCLUDED.is_closed,
		              updated_at = now()`,
		symbol,
		openTime,
		openTime.Add(time.Minute),
	); err != nil {
		ensureAPIPositivePriceConstraint(t, ctx, pool)
		t.Fatal(err)
	}
	ensureAPIPositivePriceConstraint(t, ctx, pool)
}

func dropAPIPositivePriceConstraint(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
		ALTER TABLE market_candles
		DROP CONSTRAINT IF EXISTS market_candles_positive_price_values_check`); err != nil {
		t.Fatal(err)
	}
}

func ensureAPIPositivePriceConstraint(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
		ALTER TABLE market_candles
		DROP CONSTRAINT IF EXISTS market_candles_positive_price_values_check`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		ALTER TABLE market_candles
		ADD CONSTRAINT market_candles_positive_price_values_check
		CHECK (open > 0 AND high > 0 AND low > 0 AND close > 0 AND volume >= 0)
		NOT VALID`); err != nil {
		t.Fatal(err)
	}
}

func cleanupAPIIntegrationMarket(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	symbol string,
	username string,
) {
	t.Helper()

	_, _ = pool.Exec(ctx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
	_, _ = pool.Exec(ctx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	_, _ = pool.Exec(ctx, `DELETE FROM market_instruments WHERE exchange = 'binance' AND symbol = $1`, symbol)
	_, _ = pool.Exec(ctx, `DELETE FROM operators WHERE username = $1`, username)
	ensureAPIPositivePriceConstraint(t, ctx, pool)
}
