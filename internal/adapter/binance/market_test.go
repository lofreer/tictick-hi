package binance

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestFetchCandles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/klines" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("symbol") != "BTCUSDT" || query.Get("interval") != "1m" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[[1767225600000,"100.1","101.2","99.8","100.7","12.5",1767225659999]]`))
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	candles, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Limit:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 || candles[0].Open != "100.1" || candles[0].Volume != "12.5" {
		t.Fatalf("unexpected candles: %#v", candles)
	}
}

func TestFetchCandlesConsumesKlineWeightBeforeRequest(t *testing.T) {
	limiter := &recordingRateLimiter{err: context.Canceled}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("HTTP request should not be sent when rate limit wait fails")
	}))
	defer server.Close()

	client := NewMarketClientWithOptions(MarketClientOptions{
		BaseURLs:    []string{server.URL},
		HTTPClient:  server.Client(),
		RateLimiter: limiter,
	})
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Limit:    1,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FetchCandles error = %v, want context canceled", err)
	}
	if len(limiter.weights) != 1 || limiter.weights[0] != klinesRequestWeight {
		t.Fatalf("rate limiter weights = %#v, want [%d]", limiter.weights, klinesRequestWeight)
	}
}

func TestFetchInstruments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/exchangeInfo" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT","isSpotTradingAllowed":true},{"symbol":"OLDUSDT","status":"BREAK","baseAsset":"OLD","quoteAsset":"USDT","isSpotTradingAllowed":true},{"symbol":"PERPUSDT","status":"TRADING","baseAsset":"PERP","quoteAsset":"USDT","isSpotTradingAllowed":false}]}`))
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	instruments, err := client.FetchInstruments(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 2 {
		t.Fatalf("instruments = %#v, want 2 spot instruments", instruments)
	}
	if instruments[0].Exchange != "binance" ||
		instruments[0].Symbol != "BTCUSDT" ||
		instruments[0].BaseAsset != "BTC" ||
		instruments[0].QuoteAsset != "USDT" ||
		instruments[0].Status != "active" {
		t.Fatalf("unexpected active instrument: %#v", instruments[0])
	}
	if instruments[1].Symbol != "OLDUSDT" || instruments[1].Status != "inactive" {
		t.Fatalf("unexpected inactive instrument: %#v", instruments[1])
	}
}

func TestFetchInstrumentsConsumesExchangeInfoWeightBeforeRequest(t *testing.T) {
	limiter := &recordingRateLimiter{err: context.Canceled}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("HTTP request should not be sent when rate limit wait fails")
	}))
	defer server.Close()

	client := NewMarketClientWithOptions(MarketClientOptions{
		BaseURLs:    []string{server.URL},
		HTTPClient:  server.Client(),
		RateLimiter: limiter,
	})
	_, err := client.FetchInstruments(t.Context())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FetchInstruments error = %v, want context canceled", err)
	}
	if len(limiter.weights) != 1 || limiter.weights[0] != exchangeInfoRequestWeight {
		t.Fatalf("rate limiter weights = %#v, want [%d]", limiter.weights, exchangeInfoRequestWeight)
	}
}

func TestFetchCandlesFallsBackToNextBaseURL(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer primary.Close()

	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[[1767225600000,"100.1","101.2","99.8","100.7","12.5",1767225659999]]`))
	}))
	defer secondary.Close()

	client := NewMarketClientWithBaseURLs(
		[]string{primary.URL, secondary.URL},
		secondary.Client(),
	)
	candles, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Limit:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 {
		t.Fatalf("candles = %d, want 1", len(candles))
	}
}

func TestEndpointErrorHidesQueryURL(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "https://api.binance.com/api/v3/klines?symbol=BTCUSDT&limit=500",
		Err: errors.New("EOF"),
	}

	summary := exchange.EndpointErrorSummary("https://api.binance.com", err)
	if strings.Contains(summary, "/api/v3/klines") || strings.Contains(summary, "symbol=BTCUSDT") {
		t.Fatalf("summary leaks request URL: %s", summary)
	}
	if !strings.Contains(summary, "api.binance.com") || !strings.Contains(summary, "EOF") {
		t.Fatalf("summary misses host or reason: %s", summary)
	}
}

func TestFetchCandlesMarksTransportFailuresTemporary(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	primary.Close()
	secondary := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	secondary.Close()

	client := NewMarketClientWithBaseURLs(
		[]string{primary.URL, secondary.URL},
		primary.Client(),
	)
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Limit:    1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !exchange.IsTemporaryError(err) {
		t.Fatalf("error is not temporary: %v", err)
	}
	if strings.Contains(err.Error(), "/api/v3/klines") || strings.Contains(err.Error(), "symbol=BTCUSDT") {
		t.Fatalf("error leaks request URL: %v", err)
	}
}

func TestFetchCandlesDoesNotMarkBadRequestTemporary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Limit:    1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if exchange.IsTemporaryError(err) {
		t.Fatalf("bad request should not be temporary: %v", err)
	}
}

type recordingRateLimiter struct {
	weights []int
	err     error
}

func (limiter *recordingRateLimiter) Wait(_ context.Context, weight int) error {
	limiter.weights = append(limiter.weights, weight)
	return limiter.err
}
