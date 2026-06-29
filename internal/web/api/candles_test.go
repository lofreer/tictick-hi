package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
	if result.Window.Count != 1 || result.Window.From == nil || !result.Window.From.Equal(now) {
		t.Fatalf("unexpected window metadata: %#v", result.Window)
	}
	if len(result.Candles) != 1 || result.Candles[0].Open != "100.1" {
		t.Fatalf("unexpected candles: %#v", result.Candles)
	}
}

func TestCandlesRouteReturnsInvalidHealthForHistoricalBadCandles(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository.candles = append(repository.candles, data.Candle{
		Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
		OpenTime: now, CloseTime: now.Add(time.Minute),
		Open: "0", High: "1", Low: "0", Close: "0.5", Volume: "0",
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
	if result.Source != data.CandleSourceNative || result.Health != data.CandleHealthInvalid {
		t.Fatalf("unexpected invalid metadata: %#v", result)
	}
	if len(result.Candles) != 0 || len(result.Issues) != 1 ||
		result.Issues[0].OpenTime == nil ||
		!strings.Contains(result.Issues[0].Message, "price value must be positive") {
		t.Fatalf("unexpected invalid candle payload: %#v", result)
	}
}

func TestCandlesRouteReturnsPaginationMetadata(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for index := 0; index < 3; index++ {
		openTime := start.Add(time.Duration(index) * time.Minute)
		repository.candles = append(repository.candles, data.Candle{
			Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
			Open: "100", High: "101", Low: "99", Close: "100", Volume: "1",
			IsClosed: true,
		})
	}

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		fmt.Sprintf(
			"/api/candles?exchange=binance&symbol=BTCUSDT&interval=1m&from=%s&limit=1",
			url.QueryEscape(start.Add(time.Minute).Format(time.RFC3339)),
		),
		"",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.CandleResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.Pagination.HasPrevious || !result.Pagination.HasNext {
		t.Fatalf("expected previous and next pagination: %#v", result.Pagination)
	}
	if result.Pagination.PreviousTo == nil || !result.Pagination.PreviousTo.Equal(start) {
		t.Fatalf("unexpected previous cursor: %#v", result.Pagination)
	}
	if result.Pagination.NextFrom == nil || !result.Pagination.NextFrom.Equal(start.Add(2*time.Minute)) {
		t.Fatalf("unexpected next cursor: %#v", result.Pagination)
	}
	if result.Pagination.PreviousCursor == "" || result.Pagination.NextCursor == "" {
		t.Fatalf("expected opaque cursors: %#v", result.Pagination)
	}
}

func TestCandlesRouteAcceptsOpaquePaginationCursor(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for index := 0; index < 4; index++ {
		openTime := start.Add(time.Duration(index) * time.Minute)
		repository.candles = append(repository.candles, data.Candle{
			Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
			Open: "100", High: "101", Low: "99", Close: "100", Volume: "1",
			IsClosed: true,
		})
	}
	firstFrom := start.Add(time.Minute)
	firstRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		fmt.Sprintf(
			"/api/candles?exchange=binance&symbol=BTCUSDT&interval=1m&from=%s&limit=1",
			url.QueryEscape(firstFrom.Format(time.RFC3339)),
		),
		"",
	)
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", firstRecorder.Code, firstRecorder.Body.String())
	}
	var firstResult data.CandleResult
	if err := json.NewDecoder(firstRecorder.Body).Decode(&firstResult); err != nil {
		t.Fatal(err)
	}
	if firstResult.Pagination.NextCursor == "" {
		t.Fatalf("expected next cursor: %#v", firstResult.Pagination)
	}

	nextRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/candles?exchange=binance&symbol=BTCUSDT&interval=1m&cursor="+url.QueryEscape(firstResult.Pagination.NextCursor),
		"",
	)
	if nextRecorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", nextRecorder.Code, nextRecorder.Body.String())
	}
	var nextResult data.CandleResult
	if err := json.NewDecoder(nextRecorder.Body).Decode(&nextResult); err != nil {
		t.Fatal(err)
	}
	if len(nextResult.Candles) != 1 || !nextResult.Candles[0].OpenTime.Equal(start.Add(2*time.Minute)) {
		t.Fatalf("unexpected cursor result: %#v", nextResult.Candles)
	}
}

func TestCandlesRouteRejectsCursorContextMismatch(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cursor, err := data.EncodeCandleCursor(data.NewCandleCursor(data.CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		Limit:    1,
	}, start, start, 1))
	if err != nil {
		t.Fatal(err)
	}

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/candles?exchange=binance&symbol=ETHUSDT&interval=1m&cursor="+url.QueryEscape(cursor),
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestCandlesRouteRejectsCursorMixedWithExplicitWindow(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cursor, err := data.EncodeCandleCursor(data.NewCandleCursor(data.CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		Limit:    1,
	}, start, start, 1))
	if err != nil {
		t.Fatal(err)
	}

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/candles?exchange=binance&symbol=BTCUSDT&interval=1m&from="+url.QueryEscape(start.Format(time.RFC3339))+"&cursor="+url.QueryEscape(cursor),
		"",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
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

func TestCandlesRouteRejectsExchangeSymbolMismatch(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		message string
	}{
		{
			name:    "binance hyphen symbol",
			path:    "/api/candles?exchange=binance&symbol=BTC-USDT&interval=1m",
			message: "binance symbol must use uppercase compact format such as BTCUSDT",
		},
		{
			name:    "okx compact symbol",
			path:    "/api/candles?exchange=okx&symbol=BTCUSDT&interval=1m",
			message: "okx symbol must use uppercase instrument format such as BTC-USDT",
		},
		{
			name:    "unsupported exchange",
			path:    "/api/candles?exchange=kraken&symbol=BTCUSDT&interval=1m",
			message: "exchange must be binance or okx",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, server, cookie := newAuthenticatedTestServer(t)

			recorder := serveAuthenticated(server, cookie, http.MethodGet, testCase.path, "")

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			response := decodeAPIError(t, recorder)
			if response.Code != "invalid_request" || response.Message != testCase.message {
				t.Fatalf("unexpected response: %#v", response)
			}
		})
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
