package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestCreateBacktestNormalizesDefaultStrategyParams(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	body := `{
		"name":"EMA BTC backtest",
		"exchange":"binance",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-02T00:00:00Z",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12},
		"initialBalance":"10000",
		"feeBps":"1",
		"slippageBps":"0.5",
		"triggerMode":"closed_candle"
	}`
	recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/backtests", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}

	var created data.BacktestTask
	if err := json.NewDecoder(recorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.StrategyParams["slowPeriod"] != float64(26) && created.StrategyParams["slowPeriod"] != 26 {
		t.Fatalf("slowPeriod was not defaulted: %#v", created.StrategyParams)
	}
	if created.StrategyParams["signalMode"] != "order" {
		t.Fatalf("signalMode was not defaulted: %#v", created.StrategyParams)
	}
}

func TestCreateBacktestRejectsInvalidStrategyParams(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	body := `{
		"name":"EMA BTC backtest",
		"exchange":"binance",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-02T00:00:00Z",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":1,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"initialBalance":"10000",
		"feeBps":"1",
		"slippageBps":"0.5",
		"triggerMode":"closed_candle"
	}`
	recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/backtests", body)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateTradingRejectsUnknownStrategyParams(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	body := `{
		"name":"Paper EMA",
		"type":"paper",
		"exchange":"binance",
		"accountId":"paper",
		"symbol":"BTCUSDT",
		"interval":"1m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order","unknown":true},
		"intentPolicy":{"orderIntent":"execute"}
	}`
	recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", body)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}
