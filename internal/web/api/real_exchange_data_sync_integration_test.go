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

	"github.com/lofreer/tictick-hi/internal/adapter/binance"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/datasync"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestIntegrationRealBinanceDataSyncRouteServesNativeCandles(t *testing.T) {
	if strings.TrimSpace(os.Getenv("TICTICK_REAL_EXCHANGE_SMOKE")) != "1" {
		t.Skip("set TICTICK_REAL_EXCHANGE_SMOKE=1 to run the real public exchange smoke")
	}

	store, pool, ctx := openAPIIntegrationStore(t)
	server := NewServer(store, "")

	symbol := strings.ToUpper(strings.TrimSpace(os.Getenv("TICTICK_REAL_EXCHANGE_SYMBOL")))
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	baseURL := strings.TrimSpace(os.Getenv("TICTICK_REAL_BINANCE_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://data-api.binance.vision"
	}
	username := fmt.Sprintf("api-real-sync-%d", time.Now().UTC().UnixNano())
	password := "secret123"
	end := time.Now().UTC().Truncate(time.Minute).Add(-30 * time.Minute)
	start := end.Add(-2 * time.Minute)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := contextWithShortTimeout()
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

	createBody := fmt.Sprintf(
		`{"exchange":"binance","symbol":%q,"interval":"1m","startTime":%q,"endTime":%q}`,
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
			"binance": binance.NewMarketClientWithBaseURLs([]string{baseURL}, &http.Client{Timeout: 20 * time.Second}),
		}),
		datasync.Config{
			WorkerID:          "real-binance-smoke-worker",
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

	candlesPath := "/api/candles?exchange=binance&symbol=" + url.QueryEscape(symbol) +
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

func contextWithShortTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}
