package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func serveAuthenticatedWithRequestID(
	server http.Handler,
	auth *authTestSession,
	method string,
	path string,
	body string,
	requestID string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set(requestIDHeaderName, requestID)
	request.AddCookie(auth.session)
	request.AddCookie(auth.csrf)
	if !isSafeMethod(method) {
		request.Header.Set(csrfHeaderName, auth.csrf.Value)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func TestTaskCreateRoutesPropagateRequestID(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.accounts = append(repository.accounts, data.ExchangeAccount{
		ID:               "acct_live",
		Exchange:         "binance",
		Alias:            "main",
		Enabled:          true,
		CredentialStatus: "encrypted",
	})

	dataSyncRecorder := serveAuthenticatedWithRequestID(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m"}`,
		"request-id-data",
	)
	if dataSyncRecorder.Code != http.StatusCreated {
		t.Fatalf("data sync create status = %d body = %s", dataSyncRecorder.Code, dataSyncRecorder.Body.String())
	}
	var dataSyncTask data.DataSyncTask
	if err := json.NewDecoder(dataSyncRecorder.Body).Decode(&dataSyncTask); err != nil {
		t.Fatal(err)
	}
	if dataSyncTask.RequestID != "request-id-data" {
		t.Fatalf("data sync request id = %q", dataSyncTask.RequestID)
	}

	backtestRecorder := serveAuthenticatedWithRequestID(
		server,
		auth,
		http.MethodPost,
		"/api/backtests",
		`{
			"name":"EMA BTC backtest",
			"exchange":"binance",
			"symbol":"BTCUSDT",
			"interval":"5m",
			"startTime":"2026-01-01T00:00:00Z",
			"endTime":"2026-01-02T00:00:00Z",
			"strategyId":"ema-cross",
			"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
			"initialBalance":"10000",
			"feeBps":"1",
			"slippageBps":"0.5",
			"triggerMode":"closed_candle"
		}`,
		"request-id-backtest",
	)
	if backtestRecorder.Code != http.StatusCreated {
		t.Fatalf("backtest create status = %d body = %s", backtestRecorder.Code, backtestRecorder.Body.String())
	}
	var backtestTask data.BacktestTask
	if err := json.NewDecoder(backtestRecorder.Body).Decode(&backtestTask); err != nil {
		t.Fatal(err)
	}
	if backtestTask.RequestID != "request-id-backtest" {
		t.Fatalf("backtest request id = %q", backtestTask.RequestID)
	}

	tradingRecorder := serveAuthenticatedWithRequestID(
		server,
		auth,
		http.MethodPost,
		"/api/trading/tasks",
		`{
			"name":"Paper EMA",
			"type":"paper",
			"exchange":"binance",
			"accountId":"paper",
			"symbol":"BTCUSDT",
			"interval":"5m",
			"strategyId":"ema-cross",
			"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
			"intentPolicy":{"orderIntent":"execute","notificationChannel":"default"}
		}`,
		"request-id-trading",
	)
	if tradingRecorder.Code != http.StatusCreated {
		t.Fatalf("trading create status = %d body = %s", tradingRecorder.Code, tradingRecorder.Body.String())
	}
	var tradingTask data.TradingTask
	if err := json.NewDecoder(tradingRecorder.Body).Decode(&tradingTask); err != nil {
		t.Fatal(err)
	}
	if tradingTask.RequestID != "request-id-trading" {
		t.Fatalf("trading request id = %q", tradingTask.RequestID)
	}
}
