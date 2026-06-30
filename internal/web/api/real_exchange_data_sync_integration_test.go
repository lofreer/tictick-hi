package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lofreer/tictick-hi/internal/adapter/binance"
	"github.com/lofreer/tictick-hi/internal/adapter/okx"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/datasync"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestIntegrationRealBinanceDataSyncRouteServesNativeCandles(t *testing.T) {
	runRealDataSyncRouteServesNativeCandles(t, realExchangeSmokeCase{
		exchange:        "binance",
		defaultSymbol:   "BTCUSDT",
		legacySymbolEnv: "TICTICK_REAL_EXCHANGE_SYMBOL",
		symbolEnv:       "TICTICK_REAL_BINANCE_SYMBOL",
		baseURLEnv:      "TICTICK_REAL_BINANCE_BASE_URL",
		defaultBaseURL:  "https://data-api.binance.vision",
		workerID:        "real-binance-smoke-worker",
		status:          "TRADING",
		newMarketData: func(baseURL string) exchange.MarketDataClient {
			return binance.NewMarketClientWithBaseURLs([]string{baseURL}, &http.Client{Timeout: 20 * time.Second})
		},
	})
}

func TestIntegrationRealOKXDataSyncRouteServesNativeCandles(t *testing.T) {
	if strings.TrimSpace(os.Getenv("TICTICK_REAL_OKX_SMOKE")) != "1" {
		t.Skip("set TICTICK_REAL_OKX_SMOKE=1 with TICTICK_REAL_EXCHANGE_SMOKE=1 to run the OKX public exchange smoke")
	}
	runRealDataSyncRouteServesNativeCandles(t, realExchangeSmokeCase{
		exchange:       "okx",
		defaultSymbol:  "BTC-USDT",
		symbolEnv:      "TICTICK_REAL_OKX_SYMBOL",
		baseURLEnv:     "TICTICK_REAL_OKX_BASE_URL",
		defaultBaseURL: "https://www.okx.com",
		workerID:       "real-okx-smoke-worker",
		status:         "live",
		newMarketData: func(baseURL string) exchange.MarketDataClient {
			return okx.NewMarketClientForURL(baseURL, &http.Client{Timeout: 20 * time.Second})
		},
	})
}

type realExchangeSmokeCase struct {
	exchange        string
	defaultSymbol   string
	legacySymbolEnv string
	symbolEnv       string
	baseURLEnv      string
	defaultBaseURL  string
	workerID        string
	status          string
	newMarketData   func(baseURL string) exchange.MarketDataClient
}

func runRealDataSyncRouteServesNativeCandles(t *testing.T, smoke realExchangeSmokeCase) {
	t.Helper()
	if strings.TrimSpace(os.Getenv("TICTICK_REAL_EXCHANGE_SMOKE")) != "1" {
		t.Skip("set TICTICK_REAL_EXCHANGE_SMOKE=1 to run the real public exchange smoke")
	}

	store, pool, ctx := openAPIIntegrationStore(t)
	server := NewServer(store, "")

	symbol := strings.ToUpper(strings.TrimSpace(os.Getenv(smoke.symbolEnv)))
	if symbol == "" && smoke.legacySymbolEnv != "" {
		symbol = strings.ToUpper(strings.TrimSpace(os.Getenv(smoke.legacySymbolEnv)))
	}
	if symbol == "" {
		symbol = smoke.defaultSymbol
	}
	baseURL := strings.TrimSpace(os.Getenv(smoke.baseURLEnv))
	if baseURL == "" {
		baseURL = smoke.defaultBaseURL
	}
	username := fmt.Sprintf("api-real-%s-sync-%d", smoke.exchange, time.Now().UTC().UnixNano())
	password := "secret123"
	end := time.Now().UTC().Truncate(time.Minute).Add(-30 * time.Minute)
	start := end.Add(-2 * time.Minute)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := contextWithShortTimeout()
		defer cleanupCancel()
		cleanupRealAPIIntegrationMarket(t, cleanupCtx, pool, smoke.exchange, symbol, username)
	})

	if _, _, err := store.EnsureOperator(ctx, data.CreateOperator{
		Username: username,
		Password: password,
		Enabled:  true,
	}); err != nil {
		t.Fatal(err)
	}
	auth := loginIntegrationOperator(t, server, username, password)
	upsertRealAPIIntegrationMarketInstrument(t, ctx, pool, smoke.exchange, symbol, smoke.status)

	createBody := fmt.Sprintf(
		`{"exchange":%q,"symbol":%q,"interval":"1m","startTime":%q,"endTime":%q}`,
		smoke.exchange,
		symbol,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)
	createRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/data/tasks", createBody)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create data sync task status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}
	var task data.DataSyncTask
	if err := json.NewDecoder(createRecorder.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}

	startRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/data/tasks/"+task.ID+"/sync/start", "{}")
	if startRecorder.Code != http.StatusOK {
		t.Fatalf("start data sync task status = %d body = %s", startRecorder.Code, startRecorder.Body.String())
	}

	runner := datasync.NewRunner(
		store,
		exchange.NewRegistry(map[string]exchange.MarketDataClient{
			smoke.exchange: smoke.newMarketData(baseURL),
		}),
		datasync.Config{
			WorkerID:          smoke.workerID,
			LeaseTTL:          time.Minute,
			HeartbeatInterval: time.Second,
			BatchLimit:        10,
			FetchRetries:      1,
			RetryDelay:        500 * time.Millisecond,
			RetryBackoff:      time.Second,
			MaxRetryBackoff:   10 * time.Second,
		},
	)
	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}

	after := getAPIIntegrationDataSyncTask(t, server, auth, task.ID)
	if after.Status != data.TaskStatusSucceeded || after.SyncEnabled || after.RealtimeEnabled {
		t.Fatalf("real data sync task state = %#v, want succeeded one-shot sync", after)
	}
	if after.LatestSyncedOpenTime == nil || after.LatestSyncedOpenTime.Before(start) {
		t.Fatalf("real data sync latest cursor = %#v, want at least %s", after.LatestSyncedOpenTime, start)
	}
	if after.DataHealth != data.DataSyncHealthOK {
		t.Fatalf("real data sync health = %q, want ok; task=%#v", after.DataHealth, after)
	}

	candlesPath := "/api/candles?exchange=" + url.QueryEscape(smoke.exchange) + "&symbol=" + url.QueryEscape(symbol) +
		"&interval=1m&from=" + url.QueryEscape(start.Format(time.RFC3339)) +
		"&to=" + url.QueryEscape(end.Format(time.RFC3339)) +
		"&limit=10"
	candlesRecorder := serveAuthenticated(server, auth, http.MethodGet, candlesPath, "")
	if candlesRecorder.Code != http.StatusOK {
		t.Fatalf("candles status = %d body = %s", candlesRecorder.Code, candlesRecorder.Body.String())
	}
	var result data.CandleResult
	if err := json.NewDecoder(candlesRecorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Source != data.CandleSourceNative || result.Health != data.CandleHealthOK ||
		result.RequestedInterval != "1m" || result.Window.Count != 3 || len(result.Candles) != 3 {
		t.Fatalf("real API candle result = %#v, want 3 healthy native candles", result)
	}
	if !result.Candles[0].OpenTime.Equal(start) || !result.Candles[2].OpenTime.Equal(end) {
		t.Fatalf("real API candle open times = %s..%s, want %s..%s",
			result.Candles[0].OpenTime,
			result.Candles[2].OpenTime,
			start,
			end,
		)
	}
}

func upsertRealAPIIntegrationMarketInstrument(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	exchangeName string,
	symbol string,
	exchangeStatus string,
) {
	t.Helper()

	baseAsset, quoteAsset := realAPIIntegrationMarketAssets(t, exchangeName, symbol)
	if _, err := pool.Exec(ctx, `
		INSERT INTO market_instruments (
			exchange, symbol, base_asset, quote_asset, instrument_type, status, exchange_status, search_priority, synced_at
		)
		VALUES ($1, $2, $3, $4, 'spot', 'active', $5, 0, now())
		ON CONFLICT (exchange, symbol)
		DO UPDATE SET status = EXCLUDED.status,
		              exchange_status = EXCLUDED.exchange_status,
		              synced_at = EXCLUDED.synced_at,
		              updated_at = now()`,
		exchangeName,
		symbol,
		baseAsset,
		quoteAsset,
		exchangeStatus,
	); err != nil {
		t.Fatal(err)
	}
}

func realAPIIntegrationMarketAssets(t *testing.T, exchangeName string, symbol string) (string, string) {
	t.Helper()

	switch exchangeName {
	case "binance":
		if !strings.HasSuffix(symbol, "USDT") {
			t.Fatalf("unsupported binance real smoke symbol %q", symbol)
		}
		return strings.TrimSuffix(symbol, "USDT"), "USDT"
	case "okx":
		parts := strings.Split(symbol, "-")
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			t.Fatalf("unsupported okx real smoke symbol %q", symbol)
		}
		return parts[0], parts[1]
	default:
		t.Fatalf("unsupported real smoke exchange %q", exchangeName)
		return "", ""
	}
}

func cleanupRealAPIIntegrationMarket(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	exchangeName string,
	symbol string,
	username string,
) {
	t.Helper()

	_, _ = pool.Exec(ctx, `DELETE FROM data_sync_tasks WHERE exchange = $1 AND symbol = $2`, exchangeName, symbol)
	_, _ = pool.Exec(ctx, `DELETE FROM market_candles WHERE exchange = $1 AND symbol = $2`, exchangeName, symbol)
	_, _ = pool.Exec(ctx, `DELETE FROM market_instruments WHERE exchange = $1 AND symbol = $2`, exchangeName, symbol)
	_, _ = pool.Exec(ctx, `DELETE FROM operators WHERE username = $1`, username)
	ensureAPIPositivePriceConstraint(t, ctx, pool)
}

func contextWithShortTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}
