package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateBacktestRejectsExchangeSymbolMismatch(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	body := `{
		"name":"EMA mismatch backtest",
		"exchange":"binance",
		"symbol":"BTC-USDT",
		"interval":"5m",
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-02T00:00:00Z",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"initialBalance":"10000",
		"feeBps":"1",
		"slippageBps":"0.5",
		"triggerMode":"closed_candle"
	}`
	recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/backtests", body)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "binance symbol must use uppercase compact format") {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestCreateTradingRejectsExchangeSymbolMismatch(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	body := `{
		"name":"Paper EMA mismatch",
		"type":"paper",
		"exchange":"okx",
		"accountId":"paper",
		"symbol":"BTCUSDT",
		"interval":"1m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"execute"}
	}`
	recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", body)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "okx symbol must use uppercase instrument format") {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestCreateBacktestRequiresActiveMarketInstrument(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.marketInstruments = append(
		repository.marketInstruments,
		marketInstrumentForTest("binance", "SOLUSDT", "inactive"),
	)

	body := `{
		"name":"EMA inactive backtest",
		"exchange":"binance",
		"symbol":"SOLUSDT",
		"interval":"5m",
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-02T00:00:00Z",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"initialBalance":"10000",
		"feeBps":"1",
		"slippageBps":"0.5",
		"triggerMode":"closed_candle"
	}`
	recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/backtests", body)
	assertMarketInstrumentNotActive(t, recorder)
	if len(repository.backtests) != 0 {
		t.Fatalf("inactive market backtest was persisted: %#v", repository.backtests)
	}
}

func TestCreateTradingRequiresActiveMarketInstrument(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)

	body := `{
		"name":"Paper EMA missing market",
		"type":"paper",
		"exchange":"binance",
		"accountId":"paper",
		"symbol":"MISSINGUSDT",
		"interval":"1m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"execute"}
	}`
	recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", body)
	assertMarketInstrumentNotActive(t, recorder)
	if len(repository.tradingTasks) != 0 {
		t.Fatalf("missing market trading task was persisted: %#v", repository.tradingTasks)
	}
}

func assertMarketInstrumentNotActive(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "market_instrument_not_active" ||
		response.Message != "market instrument is not active in catalog" {
		t.Fatalf("unexpected response: %#v", response)
	}
}
