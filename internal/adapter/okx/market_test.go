package okx

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
	var requestID string
	var traceparent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/market/history-candles" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("instId") != "BTC-USDT" || query.Get("bar") != "1H" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		requestID = r.Header.Get("X-Request-ID")
		traceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[["1767229200000","101","102","100","101.5","9","0","0","1"],["1767225600000","100","101","99","100.5","8","0","0","1"]]}`))
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	ctx := exchange.ContextWithRequestMetadata(
		t.Context(),
		"request-id-okx",
		"00-4BF92F3577B34DA6A3CE929D0E0E4736-00F067AA0BA902B7-01",
	)
	candles, err := client.FetchCandles(ctx, exchange.CandleRequest{
		Exchange: "okx",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
		Limit:    2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 2 || candles[0].Open != "100" || candles[1].Open != "101" {
		t.Fatalf("unexpected candles: %#v", candles)
	}
	if requestID != "request-id-okx" {
		t.Fatalf("X-Request-ID = %q, want request-id-okx", requestID)
	}
	wantTraceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	if traceparent != wantTraceparent {
		t.Fatalf("traceparent = %q, want %s", traceparent, wantTraceparent)
	}
}

func TestFetchCandlesConsumesMarketRequestBeforeHTTP(t *testing.T) {
	limiter := &recordingRateLimiter{err: context.Canceled}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("HTTP request should not be sent when rate limit wait fails")
	}))
	defer server.Close()

	client := NewMarketClientWithOptions(MarketClientOptions{
		BaseURL:     server.URL,
		HTTPClient:  server.Client(),
		RateLimiter: limiter,
	})
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "okx",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
		Limit:    2,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FetchCandles error = %v, want context canceled", err)
	}
	if len(limiter.weights) != 1 || limiter.weights[0] != marketRequestWeight {
		t.Fatalf("rate limiter weights = %#v, want [%d]", limiter.weights, marketRequestWeight)
	}
}

func TestFetchInstruments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/public/instruments" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("instType") != "SPOT" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"BTC-USDT","baseCcy":"BTC","quoteCcy":"USDT","state":"live"},{"instId":"OLD-USDT","baseCcy":"OLD","quoteCcy":"USDT","state":"suspend"}]}`))
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	instruments, err := client.FetchInstruments(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 2 {
		t.Fatalf("instruments = %#v, want 2", instruments)
	}
	if instruments[0].Exchange != "okx" ||
		instruments[0].Symbol != "BTC-USDT" ||
		instruments[0].BaseAsset != "BTC" ||
		instruments[0].QuoteAsset != "USDT" ||
		instruments[0].Status != "active" ||
		instruments[0].ExchangeStatus != "live" {
		t.Fatalf("unexpected active instrument: %#v", instruments[0])
	}
	if instruments[1].Symbol != "OLD-USDT" || instruments[1].Status != "inactive" || instruments[1].ExchangeStatus != "suspend" {
		t.Fatalf("unexpected inactive instrument: %#v", instruments[1])
	}
}

func TestFetchInstrumentsConsumesMarketRequestBeforeHTTP(t *testing.T) {
	limiter := &recordingRateLimiter{err: context.Canceled}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("HTTP request should not be sent when rate limit wait fails")
	}))
	defer server.Close()

	client := NewMarketClientWithOptions(MarketClientOptions{
		BaseURL:     server.URL,
		HTTPClient:  server.Client(),
		RateLimiter: limiter,
	})
	_, err := client.FetchInstruments(t.Context())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FetchInstruments error = %v, want context canceled", err)
	}
	if len(limiter.weights) != 1 || limiter.weights[0] != marketRequestWeight {
		t.Fatalf("rate limiter weights = %#v, want [%d]", limiter.weights, marketRequestWeight)
	}
}

func TestFetchCandlesMarksServiceUnavailableTemporary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "okx",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
		Limit:    2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !exchange.IsTemporaryError(err) {
		t.Fatalf("error is not temporary: %v", err)
	}
}

func TestFetchCandlesMarksRateLimitCodeTemporary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"50011","msg":"Requests too frequent","data":[]}`))
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "okx",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
		Limit:    2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !exchange.IsTemporaryError(err) {
		t.Fatalf("rate limit code should be temporary: %v", err)
	}
}

func TestFetchCandlesPreservesRateLimitCodeRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "13")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"50011","msg":"Requests too frequent","data":[]}`))
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "okx",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
		Limit:    2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	retryAfter, ok := exchange.RetryAfter(err)
	if !ok || retryAfter != 13*time.Second {
		t.Fatalf("RetryAfter = %s, %t; want 13s, true", retryAfter, ok)
	}
}

func TestFetchCandlesPreservesRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "8")
		http.Error(w, "too many requests", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "okx",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
		Limit:    2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	retryAfter, ok := exchange.RetryAfter(err)
	if !ok || retryAfter != 8*time.Second {
		t.Fatalf("RetryAfter = %s, %t; want 8s, true", retryAfter, ok)
	}
}

func TestFetchCandlesDoesNotMarkInstrumentCodeTemporary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"51001","msg":"Instrument ID does not exist","data":[]}`))
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "okx",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
		Limit:    2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if exchange.IsTemporaryError(err) {
		t.Fatalf("instrument code should not be temporary: %v", err)
	}
}

func TestFetchInstrumentsPreservesRateLimitCodeRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "21")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"50011","msg":"Requests too frequent","data":[]}`))
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	_, err := client.FetchInstruments(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
	retryAfter, ok := exchange.RetryAfter(err)
	if !ok || retryAfter != 21*time.Second {
		t.Fatalf("RetryAfter = %s, %t; want 21s, true", retryAfter, ok)
	}
}

func TestFetchCandlesDoesNotMarkBadRequestTemporary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	_, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
		Exchange: "okx",
		Symbol:   "BTCUSDT",
		Interval: "1h",
		From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
		Limit:    2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if exchange.IsTemporaryError(err) {
		t.Fatalf("bad request should not be temporary: %v", err)
	}
}

func TestEndpointErrorHidesQueryURL(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "https://www.okx.com/api/v5/market/history-candles?instId=BTC-USDT&limit=100",
		Err: errors.New("EOF"),
	}

	summary := exchange.EndpointErrorSummary("https://www.okx.com", err)
	if strings.Contains(summary, "/api/v5/market") || strings.Contains(summary, "instId=BTC-USDT") {
		t.Fatalf("summary leaks request URL: %s", summary)
	}
	if !strings.Contains(summary, "www.okx.com") || !strings.Contains(summary, "EOF") {
		t.Fatalf("summary misses host or reason: %s", summary)
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
