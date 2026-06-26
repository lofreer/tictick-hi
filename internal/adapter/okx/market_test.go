package okx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestFetchCandles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/market/history-candles" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("instId") != "BTC-USDT" || query.Get("bar") != "1H" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[["1767229200000","101","102","100","101.5","9","0","0","1"],["1767225600000","100","101","99","100.5","8","0","0","1"]]}`))
	}))
	defer server.Close()

	client := NewMarketClientForURL(server.URL, server.Client())
	candles, err := client.FetchCandles(t.Context(), exchange.CandleRequest{
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
}
