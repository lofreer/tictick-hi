package api

import (
	"encoding/json"
	"net/http"
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
