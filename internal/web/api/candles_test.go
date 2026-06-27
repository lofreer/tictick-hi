package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestCandlesRouteReturnsMetadata(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository.candles = append(repository.candles, data.Candle{
		Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
		OpenTime: now, CloseTime: now.Add(time.Minute),
		Open: "100.1", High: "101.2", Low: "99.9", Close: "100.8", Volume: "12.5",
		IsClosed: true,
	})

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/candles?exchange=binance&symbol=BTCUSDT&interval=1m",
		"",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.CandleResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Source != data.CandleSourceNative || result.Health != data.CandleHealthOK {
		t.Fatalf("unexpected metadata: %#v", result)
	}
	if len(result.Candles) != 1 || result.Candles[0].Open != "100.1" {
		t.Fatalf("unexpected candles: %#v", result.Candles)
	}
}

func TestCandlesRouteRejectsOversizedLimit(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		fmt.Sprintf("/api/candles?exchange=binance&symbol=BTCUSDT&interval=1m&limit=%d", data.MaxCandleLimit+1),
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestCandlesRouteRejectsInvertedRange(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		candlesPath("1m", "2026-01-02T00:00:00Z", "2026-01-01T00:00:00Z"),
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestCandlesRouteRejectsOversizedRange(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		candlesPath("1m", "2026-01-01T00:00:00Z", "2026-01-04T12:00:00Z"),
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestCandlesRouteRejectsUnsupportedInterval(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/candles?exchange=binance&symbol=BTCUSDT&interval=tick",
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func candlesPath(interval string, from string, to string) string {
	return fmt.Sprintf(
		"/api/candles?exchange=binance&symbol=BTCUSDT&interval=%s&from=%s&to=%s",
		url.QueryEscape(interval),
		url.QueryEscape(from),
		url.QueryEscape(to),
	)
}
