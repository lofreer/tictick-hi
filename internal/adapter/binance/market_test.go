package binance

import (
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

	summary := endpointError("https://api.binance.com", err)
	if strings.Contains(summary, "/api/v3/klines") || strings.Contains(summary, "symbol=BTCUSDT") {
		t.Fatalf("summary leaks request URL: %s", summary)
	}
	if !strings.Contains(summary, "api.binance.com") || !strings.Contains(summary, "EOF") {
		t.Fatalf("summary misses host or reason: %s", summary)
	}
}
