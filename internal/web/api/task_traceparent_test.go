package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func serveAuthenticatedWithTraceParent(
	server http.Handler,
	auth *authTestSession,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set(traceparentHeaderName, validTestTraceparent)
	request.AddCookie(auth.session)
	request.AddCookie(auth.csrf)
	if !isSafeMethod(method) {
		request.Header.Set(csrfHeaderName, auth.csrf.Value)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func TestTaskCreateRoutesPropagateTraceParent(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.accounts = append(repository.accounts, data.ExchangeAccount{
		ID:               "acct_live",
		Exchange:         "binance",
		Alias:            "main",
		Enabled:          true,
		CredentialStatus: "encrypted",
	})

	dataSyncRecorder := serveAuthenticatedWithTraceParent(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m"}`,
	)
	if dataSyncRecorder.Code != http.StatusCreated {
		t.Fatalf("data sync create status = %d body = %s", dataSyncRecorder.Code, dataSyncRecorder.Body.String())
	}
	var dataSyncTask data.DataSyncTask
	if err := json.NewDecoder(dataSyncRecorder.Body).Decode(&dataSyncTask); err != nil {
		t.Fatal(err)
	}
	if dataSyncTask.TraceParent != validTestTraceparent {
		t.Fatalf("data sync traceparent = %q", dataSyncTask.TraceParent)
	}

	backtestRecorder := serveAuthenticatedWithTraceParent(
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
	)
	if backtestRecorder.Code != http.StatusCreated {
		t.Fatalf("backtest create status = %d body = %s", backtestRecorder.Code, backtestRecorder.Body.String())
	}
	var backtestTask data.BacktestTask
	if err := json.NewDecoder(backtestRecorder.Body).Decode(&backtestTask); err != nil {
		t.Fatal(err)
	}
	if backtestTask.TraceParent != validTestTraceparent {
		t.Fatalf("backtest traceparent = %q", backtestTask.TraceParent)
	}

	tradingRecorder := serveAuthenticatedWithTraceParent(
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
	)
	if tradingRecorder.Code != http.StatusCreated {
		t.Fatalf("trading create status = %d body = %s", tradingRecorder.Code, tradingRecorder.Body.String())
	}
	var tradingTask data.TradingTask
	if err := json.NewDecoder(tradingRecorder.Body).Decode(&tradingTask); err != nil {
		t.Fatal(err)
	}
	if tradingTask.TraceParent != validTestTraceparent {
		t.Fatalf("trading traceparent = %q", tradingTask.TraceParent)
	}
}

func TestRepairRoutesPropagateTraceParent(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	source := data.DataSyncTask{
		ID:        "dst_source_traceparent",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		Status:    data.TaskStatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.tasks = append(repository.tasks, source)

	firstGap := data.CandleGap{
		From:           time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC),
		To:             time.Date(2026, 1, 1, 0, 3, 0, 0, time.UTC),
		MissingCandles: 1,
	}
	secondGap := data.CandleGap{
		From:           time.Date(2026, 1, 1, 0, 4, 0, 0, time.UTC),
		To:             time.Date(2026, 1, 1, 0, 5, 0, 0, time.UTC),
		MissingCandles: 1,
	}
	repository.tasks[0].GapSummary = &data.DataSyncGapSummary{
		Count:    2,
		FirstGap: &firstGap,
	}
	repository.taskGapDetails[source.ID] = data.DataSyncGapList{
		TaskID:        source.ID,
		Gaps:          []data.CandleGap{firstGap, secondGap},
		TotalCount:    2,
		ReturnedCount: 2,
		RepairLimit:   20,
	}
	invalidOpenTime := time.Date(2026, 1, 1, 0, 8, 0, 0, time.UTC)
	repository.taskInvalidDetails[source.ID] = data.DataSyncInvalidIssueList{
		TaskID: source.ID,
		Issues: []data.CandleIssue{{
			Code:     data.CandleIssueInvalidOpenPrice,
			Message:  "open price value must be positive",
			OpenTime: &invalidOpenTime,
		}},
		TotalCount:    1,
		ReturnedCount: 1,
		IssueLimit:    20,
	}

	batchRecorder := serveAuthenticatedWithTraceParent(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks/"+source.ID+"/repair-gaps",
		"",
	)
	assertCreatedRepairTaskTraceParents(t, batchRecorder, 1)

	singleRecorder := serveAuthenticatedWithTraceParent(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks/"+source.ID+"/repair-gap",
		`{"from":"2026-01-01T00:04:00Z","to":"2026-01-01T00:05:00Z"}`,
	)
	assertCreatedRepairTaskTraceParents(t, singleRecorder, 1)

	invalidRecorder := serveAuthenticatedWithTraceParent(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks/"+source.ID+"/repair-invalid-issues",
		`{}`,
	)
	assertCreatedRepairTaskTraceParents(t, invalidRecorder, 1)

	appendRequestIDTestCandle(repository, time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC), "100")
	appendRequestIDTestCandle(repository, time.Date(2026, 1, 1, 1, 1, 0, 0, time.UTC), "100")
	appendRequestIDTestCandle(repository, time.Date(2026, 1, 1, 1, 3, 0, 0, time.UTC), "100")
	marketGapRecorder := serveAuthenticatedWithTraceParent(
		server,
		auth,
		http.MethodPost,
		"/api/market/candle-gaps/repair",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m","from":"2026-01-01T01:02:00Z","to":"2026-01-01T01:03:00Z"}`,
	)
	assertCreatedRepairTaskTraceParents(t, marketGapRecorder, 1)

	appendRequestIDTestCandle(repository, time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC), "100")
	appendRequestIDTestCandle(repository, time.Date(2026, 1, 1, 2, 1, 0, 0, time.UTC), "100")
	appendRequestIDTestCandle(repository, time.Date(2026, 1, 1, 2, 3, 0, 0, time.UTC), "100")
	appendRequestIDTestCandle(repository, time.Date(2026, 1, 1, 2, 6, 0, 0, time.UTC), "100")
	marketBatchRecorder := serveAuthenticatedWithTraceParent(
		server,
		auth,
		http.MethodPost,
		"/api/market/candle-gaps/repair-batch",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m","gaps":[{"from":"2026-01-01T02:02:00Z","to":"2026-01-01T02:03:00Z"},{"from":"2026-01-01T02:04:00Z","to":"2026-01-01T02:06:00Z"}]}`,
	)
	assertCreatedRepairTaskTraceParents(t, marketBatchRecorder, 2)

	invalidMarketOpen := time.Date(2026, 1, 1, 3, 2, 0, 0, time.UTC)
	appendRequestIDTestCandle(repository, invalidMarketOpen, "0")
	marketInvalidRecorder := serveAuthenticatedWithTraceParent(
		server,
		auth,
		http.MethodPost,
		"/api/market/candle-invalid-issues/repair",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m","openTimes":["2026-01-01T03:02:00Z"]}`,
	)
	assertCreatedRepairTaskTraceParents(t, marketInvalidRecorder, 1)
}

func assertCreatedRepairTaskTraceParents(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantCount int,
) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("repair status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.DataSyncGapRepairResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.CreatedTasks) != wantCount {
		t.Fatalf("created repair tasks = %#v, want %d", result.CreatedTasks, wantCount)
	}
	for _, task := range result.CreatedTasks {
		if task.TraceParent != validTestTraceparent {
			t.Fatalf("repair task traceparent = %q, want %q: %#v", task.TraceParent, validTestTraceparent, task)
		}
	}
}
