package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestMarketInstrumentRoutesRequireAuthentication(t *testing.T) {
	server := NewServer(newFakeRepository(), "")

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/market/instruments?exchange=binance", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "unauthorized" {
		t.Fatalf("unexpected auth response: %#v", response)
	}
}

func TestMarketInstrumentRoutesSearchCatalog(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.marketInstruments = []data.MarketInstrument{
		{Exchange: "binance", Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
		{Exchange: "binance", Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
		{Exchange: "binance", Symbol: "SOLBTC", BaseAsset: "SOL", QuoteAsset: "BTC", InstrumentType: "spot", Status: "inactive"},
		{Exchange: "okx", Symbol: "SOL-USDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
	}

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/market/instruments?exchange=binance&q=sol&limit=100", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var instruments []data.MarketInstrument
	if err := json.NewDecoder(recorder.Body).Decode(&instruments); err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 1 || instruments[0].Symbol != "SOLUSDT" {
		t.Fatalf("instruments = %#v, want SOLUSDT only", instruments)
	}

	allRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/market/instruments?exchange=binance&q=solbtc&status=all&limit=1", "")
	if allRecorder.Code != http.StatusOK {
		t.Fatalf("all status = %d body = %s", allRecorder.Code, allRecorder.Body.String())
	}
	var allInstruments []data.MarketInstrument
	if err := json.NewDecoder(allRecorder.Body).Decode(&allInstruments); err != nil {
		t.Fatal(err)
	}
	if len(allInstruments) != 1 || allInstruments[0].Symbol != "SOLBTC" || allInstruments[0].Status != "inactive" {
		t.Fatalf("all instruments = %#v, want inactive SOLBTC", allInstruments)
	}

	inactiveRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/market/instruments?exchange=binance&q=sol&status=inactive", "")
	if inactiveRecorder.Code != http.StatusOK {
		t.Fatalf("inactive status = %d body = %s", inactiveRecorder.Code, inactiveRecorder.Body.String())
	}
	var inactiveInstruments []data.MarketInstrument
	if err := json.NewDecoder(inactiveRecorder.Body).Decode(&inactiveInstruments); err != nil {
		t.Fatal(err)
	}
	if len(inactiveInstruments) != 1 || inactiveInstruments[0].Symbol != "SOLBTC" {
		t.Fatalf("inactive instruments = %#v, want SOLBTC only", inactiveInstruments)
	}
}

func TestMarketCandleGapRouteScansPersistedHistory(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	start := time.Date(2026, 6, 27, 3, 0, 0, 0, time.UTC)
	for _, minute := range []int{0, 1, 3, 6} {
		openTime := start.Add(time.Duration(minute) * time.Minute)
		repository.candles = append(repository.candles, data.Candle{
			Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
			Open: "100", High: "101", Low: "99", Close: "100", Volume: "1",
			IsClosed: true,
		})
	}

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodGet,
		"/api/market/candle-gaps?exchange=binance&symbol=BTCUSDT&interval=1m&limit=1",
		"",
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}

	var scan data.MarketCandleGapScan
	if err := json.NewDecoder(recorder.Body).Decode(&scan); err != nil {
		t.Fatal(err)
	}
	if scan.Exchange != "binance" || scan.Symbol != "BTCUSDT" || scan.Interval != "1m" {
		t.Fatalf("unexpected scan identity: %#v", scan)
	}
	if scan.Window.Count != 4 || scan.Window.From == nil || !scan.Window.From.Equal(start) ||
		scan.Window.To == nil || !scan.Window.To.Equal(start.Add(6*time.Minute)) {
		t.Fatalf("unexpected scan window: %#v", scan.Window)
	}
	if !scan.Limited || scan.TotalCount != 2 || scan.ReturnedCount != 1 || len(scan.Gaps) != 1 {
		t.Fatalf("unexpected gap metadata: %#v", scan)
	}
	if !scan.Gaps[0].From.Equal(start.Add(2*time.Minute)) ||
		!scan.Gaps[0].To.Equal(start.Add(3*time.Minute)) ||
		scan.Gaps[0].MissingCandles != 1 {
		t.Fatalf("unexpected first gap: %#v", scan.Gaps[0])
	}
}

func TestMarketCandleInvalidIssueRouteScansPersistedHistory(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	start := time.Date(2026, 6, 27, 3, 0, 0, 0, time.UTC)
	for _, item := range []struct {
		minute int
		open   string
		close  string
	}{
		{minute: 0, open: "100", close: "100"},
		{minute: 1, open: "0", close: "100"},
		{minute: 2, open: "100", close: "0"},
		{minute: 3, open: "100", close: "100"},
	} {
		openTime := start.Add(time.Duration(item.minute) * time.Minute)
		repository.candles = append(repository.candles, data.Candle{
			Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
			Open: item.open, High: "101", Low: "99", Close: item.close, Volume: "1",
			IsClosed: true,
		})
	}

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodGet,
		"/api/market/candle-invalid-issues?exchange=binance&symbol=BTCUSDT&interval=1m&limit=1",
		"",
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}

	var scan data.MarketCandleInvalidIssueScan
	if err := json.NewDecoder(recorder.Body).Decode(&scan); err != nil {
		t.Fatal(err)
	}
	if scan.Exchange != "binance" || scan.Symbol != "BTCUSDT" || scan.Interval != "1m" {
		t.Fatalf("unexpected scan identity: %#v", scan)
	}
	if scan.Window.Count != 4 || scan.Window.From == nil || !scan.Window.From.Equal(start) ||
		scan.Window.To == nil || !scan.Window.To.Equal(start.Add(3*time.Minute)) {
		t.Fatalf("unexpected scan window: %#v", scan.Window)
	}
	if !scan.Limited || scan.TotalCount != 2 || scan.ReturnedCount != 1 || len(scan.Issues) != 1 {
		t.Fatalf("unexpected invalid issue metadata: %#v", scan)
	}
	if scan.Issues[0].OpenTime == nil || !scan.Issues[0].OpenTime.Equal(start.Add(time.Minute)) ||
		scan.Issues[0].Code != data.CandleIssueInvalidOpenPrice ||
		scan.Issues[0].Message != "open price value must be positive" {
		t.Fatalf("unexpected first issue: %#v", scan.Issues[0])
	}

	invalidLimit := serveAuthenticated(
		server,
		auth,
		http.MethodGet,
		"/api/market/candle-invalid-issues?exchange=binance&symbol=BTCUSDT&interval=1m&limit=101",
		"",
	)
	if invalidLimit.Code != http.StatusBadRequest {
		t.Fatalf("invalid limit status = %d body = %s", invalidLimit.Code, invalidLimit.Body.String())
	}
}

func TestMarketCandleGapRepairRouteQueuesSyncTask(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	start := time.Date(2026, 6, 27, 4, 0, 0, 0, time.UTC)
	for _, minute := range []int{0, 1, 3} {
		openTime := start.Add(time.Duration(minute) * time.Minute)
		repository.candles = append(repository.candles, data.Candle{
			Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
			Open: "100", High: "101", Low: "99", Close: "100", Volume: "1",
			IsClosed: true,
		})
	}

	body := `{"exchange":"binance","symbol":"btcusdt","interval":"1m","from":"2026-06-27T04:02:00Z","to":"2026-06-27T04:03:00Z"}`
	missingCSRF := serveAuthenticatedWithoutCSRF(server, auth, http.MethodPost, "/api/market/candle-gaps/repair", body)
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status = %d body = %s", missingCSRF.Code, missingCSRF.Body.String())
	}

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/candle-gaps/repair", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("repair status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.DataSyncGapRepairResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != "" || result.SkippedExisting != 0 || result.TotalCount != 1 || result.RepairLimit != 1 {
		t.Fatalf("unexpected repair metadata: %#v", result)
	}
	if len(result.CreatedTasks) != 1 {
		t.Fatalf("created repair tasks = %#v, want one", result.CreatedTasks)
	}
	task := result.CreatedTasks[0]
	if task.RepairSourceTaskID != "" || task.StartTime == nil || !task.StartTime.Equal(start.Add(2*time.Minute)) ||
		task.EndTime == nil || !task.EndTime.Equal(start.Add(3*time.Minute)) ||
		!task.SyncEnabled || task.RealtimeEnabled || task.Status != data.TaskStatusPending {
		t.Fatalf("unexpected repair task: %#v", task)
	}

	duplicateRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/candle-gaps/repair", body)
	if duplicateRecorder.Code != http.StatusOK {
		t.Fatalf("duplicate status = %d body = %s", duplicateRecorder.Code, duplicateRecorder.Body.String())
	}
	var duplicate data.DataSyncGapRepairResult
	if err := json.NewDecoder(duplicateRecorder.Body).Decode(&duplicate); err != nil {
		t.Fatal(err)
	}
	if len(duplicate.CreatedTasks) != 0 || duplicate.SkippedExisting != 1 {
		t.Fatalf("duplicate repair result = %#v, want skipped existing", duplicate)
	}

	notGap := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/market/candle-gaps/repair",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m","from":"2026-06-27T04:01:00Z","to":"2026-06-27T04:02:00Z"}`,
	)
	if notGap.Code != http.StatusNotFound {
		t.Fatalf("not gap status = %d body = %s", notGap.Code, notGap.Body.String())
	}
}

func TestMarketCandleGapRepairRouteRequiresActiveMarketInstrument(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	start := time.Date(2026, 6, 27, 6, 0, 0, 0, time.UTC)
	repository.marketInstruments = []data.MarketInstrument{marketInstrumentForTest("binance", "BTCUSDT", "inactive")}
	for _, minute := range []int{0, 1, 3} {
		openTime := start.Add(time.Duration(minute) * time.Minute)
		repository.candles = append(repository.candles, data.Candle{
			Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
			Open: "100", High: "101", Low: "99", Close: "100", Volume: "1",
			IsClosed: true,
		})
	}

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/market/candle-gaps/repair",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m","from":"2026-06-27T06:02:00Z","to":"2026-06-27T06:03:00Z"}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "market_instrument_not_active" ||
		response.Message != "market instrument is inactive in catalog" {
		t.Fatalf("unexpected error response: %#v", response)
	}
	if len(repository.tasks) != 0 {
		t.Fatalf("repair created tasks for inactive market: %#v", repository.tasks)
	}
}

func TestMarketCandleRepairRoutesRejectUnsupportedDataSyncInterval(t *testing.T) {
	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "single gap",
			path: "/api/market/candle-gaps/repair",
			body: `{"exchange":"binance","symbol":"BTCUSDT","interval":"2m","from":"2026-06-27T04:02:00Z","to":"2026-06-27T04:04:00Z"}`,
		},
		{
			name: "batch gaps",
			path: "/api/market/candle-gaps/repair-batch",
			body: `{"exchange":"binance","symbol":"BTCUSDT","interval":"2m","gaps":[{"from":"2026-06-27T04:02:00Z","to":"2026-06-27T04:04:00Z"}]}`,
		},
		{
			name: "invalid issues",
			path: "/api/market/candle-invalid-issues/repair",
			body: `{"exchange":"binance","symbol":"BTCUSDT","interval":"2m","openTimes":["2026-06-27T04:02:00Z"]}`,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			repository, server, auth := newAuthenticatedTestServer(t)
			recorder := serveAuthenticated(server, auth, http.MethodPost, testCase.path, testCase.body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			response := decodeAPIError(t, recorder)
			if response.Code != "invalid_request" || response.Message != `unsupported data sync interval "2m"` {
				t.Fatalf("unexpected response: %#v", response)
			}
			if len(repository.tasks) != 0 {
				t.Fatalf("repair route created tasks for unsupported interval: %#v", repository.tasks)
			}
		})
	}
}

func TestMarketCandleGapBatchRepairRouteQueuesReturnedGaps(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	start := time.Date(2026, 6, 27, 5, 0, 0, 0, time.UTC)
	for _, minute := range []int{0, 1, 3, 6} {
		openTime := start.Add(time.Duration(minute) * time.Minute)
		repository.candles = append(repository.candles, data.Candle{
			Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
			Open: "100", High: "101", Low: "99", Close: "100", Volume: "1",
			IsClosed: true,
		})
	}

	body := `{"exchange":"binance","symbol":"btcusdt","interval":"1m","gaps":[{"from":"2026-06-27T05:02:00Z","to":"2026-06-27T05:03:00Z"},{"from":"2026-06-27T05:04:00Z","to":"2026-06-27T05:06:00Z"}]}`
	missingCSRF := serveAuthenticatedWithoutCSRF(server, auth, http.MethodPost, "/api/market/candle-gaps/repair-batch", body)
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status = %d body = %s", missingCSRF.Code, missingCSRF.Body.String())
	}

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/candle-gaps/repair-batch", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("batch repair status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.DataSyncGapRepairResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != "" || result.SkippedExisting != 0 || result.TotalCount != 2 || result.RepairLimit != data.MaxMarketCandleGapScanLimit {
		t.Fatalf("unexpected batch repair metadata: %#v", result)
	}
	if len(result.CreatedTasks) != 2 {
		t.Fatalf("created repair tasks = %#v, want two", result.CreatedTasks)
	}
	if result.CreatedTasks[0].StartTime == nil || !result.CreatedTasks[0].StartTime.Equal(start.Add(2*time.Minute)) ||
		result.CreatedTasks[1].StartTime == nil || !result.CreatedTasks[1].StartTime.Equal(start.Add(4*time.Minute)) {
		t.Fatalf("unexpected repair task windows: %#v", result.CreatedTasks)
	}

	duplicateRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/candle-gaps/repair-batch", body)
	if duplicateRecorder.Code != http.StatusOK {
		t.Fatalf("duplicate status = %d body = %s", duplicateRecorder.Code, duplicateRecorder.Body.String())
	}
	var duplicate data.DataSyncGapRepairResult
	if err := json.NewDecoder(duplicateRecorder.Body).Decode(&duplicate); err != nil {
		t.Fatal(err)
	}
	if len(duplicate.CreatedTasks) != 0 || duplicate.SkippedExisting != 2 {
		t.Fatalf("duplicate batch repair result = %#v, want skipped existing", duplicate)
	}

	notGap := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/market/candle-gaps/repair-batch",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m","gaps":[{"from":"2026-06-27T05:01:00Z","to":"2026-06-27T05:02:00Z"}]}`,
	)
	if notGap.Code != http.StatusNotFound {
		t.Fatalf("not gap status = %d body = %s", notGap.Code, notGap.Body.String())
	}
}

func TestMarketCandleGapBatchRepairRouteRequiresActiveMarketInstrument(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	start := time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)
	repository.marketInstruments = nil
	for _, minute := range []int{0, 1, 3, 6} {
		openTime := start.Add(time.Duration(minute) * time.Minute)
		repository.candles = append(repository.candles, data.Candle{
			Exchange: "binance", Symbol: "ETHUSDT", Interval: "1m",
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
			Open: "100", High: "101", Low: "99", Close: "100", Volume: "1",
			IsClosed: true,
		})
	}

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/market/candle-gaps/repair-batch",
		`{"exchange":"binance","symbol":"ETHUSDT","interval":"1m","gaps":[{"from":"2026-06-27T07:02:00Z","to":"2026-06-27T07:03:00Z"},{"from":"2026-06-27T07:04:00Z","to":"2026-06-27T07:06:00Z"}]}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "market_instrument_not_active" ||
		response.Message != "market instrument is missing from catalog" {
		t.Fatalf("unexpected error response: %#v", response)
	}
	if len(repository.tasks) != 0 {
		t.Fatalf("batch repair created tasks for missing market: %#v", repository.tasks)
	}
}

func TestMarketInstrumentSyncRouteRefreshesCatalog(t *testing.T) {
	repository := newFakeRepository()
	repository.marketInstruments = []data.MarketInstrument{
		{Exchange: "binance", Symbol: "OLDUSDT", BaseAsset: "OLD", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
	}
	server := NewServerWithConfig(repository, Config{
		InstrumentClients: map[string]exchange.InstrumentClient{
			"binance": fakeInstrumentClient{instruments: []data.MarketInstrument{
				{Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
				{Symbol: "DELISTUSDT", BaseAsset: "DELIST", QuoteAsset: "USDT", InstrumentType: "spot", Status: "inactive"},
			}},
		},
	})
	auth := loginTestOperator(t, server)

	missingCSRF := serveAuthenticatedWithoutCSRF(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status = %d body = %s", missingCSRF.Code, missingCSRF.Body.String())
	}

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("sync status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.MarketInstrumentSyncResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Exchange != "binance" || result.ActiveCount != 1 || result.InactiveCount != 1 ||
		result.PausedDataSyncTaskCount != 0 || result.RestoredDataSyncTaskCount != 0 {
		t.Fatalf("unexpected sync result: %#v", result)
	}

	listRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/market/instruments?exchange=binance&q=sol", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var instruments []data.MarketInstrument
	if err := json.NewDecoder(listRecorder.Body).Decode(&instruments); err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 1 || instruments[0].Symbol != "SOLUSDT" {
		t.Fatalf("instruments = %#v, want SOLUSDT only", instruments)
	}
}

func TestMarketInstrumentSyncRouteReportsRestoredDataSyncTasks(t *testing.T) {
	repository := newFakeRepository()
	symbol := "RESTOREUSDT"
	repository.marketInstruments = []data.MarketInstrument{
		{Exchange: "binance", Symbol: symbol, BaseAsset: "RESTORE", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
	}
	repository.tasks = []data.DataSyncTask{{
		ID:              "dst_restore",
		Exchange:        "binance",
		Symbol:          symbol,
		Interval:        "1m",
		SyncEnabled:     true,
		RealtimeEnabled: true,
		Status:          data.TaskStatusRunning,
	}}
	client := &mutableInstrumentClient{instruments: []data.MarketInstrument{{
		Symbol:         symbol,
		BaseAsset:      "RESTORE",
		QuoteAsset:     "USDT",
		InstrumentType: "spot",
		Status:         "inactive",
		ExchangeStatus: "BREAK",
	}}}
	server := NewServerWithConfig(repository, Config{
		InstrumentClients: map[string]exchange.InstrumentClient{"binance": client},
	})
	auth := loginTestOperator(t, server)

	pauseRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if pauseRecorder.Code != http.StatusOK {
		t.Fatalf("pause status = %d body = %s", pauseRecorder.Code, pauseRecorder.Body.String())
	}
	var pauseResult data.MarketInstrumentSyncResult
	if err := json.NewDecoder(pauseRecorder.Body).Decode(&pauseResult); err != nil {
		t.Fatal(err)
	}
	if pauseResult.PausedDataSyncTaskCount != 1 || pauseResult.RestoredDataSyncTaskCount != 0 {
		t.Fatalf("pause result = %#v, want one paused and zero restored", pauseResult)
	}

	client.instruments = []data.MarketInstrument{{
		Symbol:         symbol,
		BaseAsset:      "RESTORE",
		QuoteAsset:     "USDT",
		InstrumentType: "spot",
		Status:         "active",
		ExchangeStatus: "TRADING",
	}}
	restoreRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if restoreRecorder.Code != http.StatusOK {
		t.Fatalf("restore status = %d body = %s", restoreRecorder.Code, restoreRecorder.Body.String())
	}
	var restoreResult data.MarketInstrumentSyncResult
	if err := json.NewDecoder(restoreRecorder.Body).Decode(&restoreResult); err != nil {
		t.Fatal(err)
	}
	if restoreResult.PausedDataSyncTaskCount != 0 || restoreResult.RestoredDataSyncTaskCount != 1 {
		t.Fatalf("restore result = %#v, want zero paused and one restored", restoreResult)
	}
	task := repository.tasks[0]
	if !task.SyncEnabled || !task.RealtimeEnabled || task.Status != data.TaskStatusRunning || task.LastError != "" {
		t.Fatalf("restored task = %#v, want original sync/realtime expectations", task)
	}
}

func TestMarketInstrumentSyncRouteRejectsUnavailableClient(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "request_failed" {
		t.Fatalf("unexpected error response: %#v", response)
	}
}

func TestMarketInstrumentSyncStatusRouteListsStatuses(t *testing.T) {
	repository := newFakeRepository()
	lastAttempt := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	lastSuccess := lastAttempt.Add(-time.Hour)
	repository.marketSyncStatuses = []data.MarketInstrumentSyncStatus{{
		Exchange:      "okx",
		LastAttemptAt: lastAttempt,
		LastSuccessAt: &lastSuccess,
		LastError:     "okx instruments temporary unavailable: www.okx.com: EOF",
		UpdatedAt:     lastAttempt,
	}}
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/market/instruments/status", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var statuses []data.MarketInstrumentSyncStatus
	if err := json.NewDecoder(recorder.Body).Decode(&statuses); err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].Exchange != "okx" || statuses[0].LastError == "" {
		t.Fatalf("unexpected statuses: %#v", statuses)
	}
}

func TestMarketInstrumentSyncRouteRecordsFetchFailure(t *testing.T) {
	repository := newFakeRepository()
	server := NewServerWithConfig(repository, Config{
		InstrumentClients: map[string]exchange.InstrumentClient{
			"okx": fakeInstrumentClient{err: errors.New("okx instruments temporary unavailable: www.okx.com: EOF")},
		},
	})
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=okx", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("sync status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(repository.marketSyncFailures) != 1 {
		t.Fatalf("market sync failures = %#v, want one", repository.marketSyncFailures)
	}
	failure := repository.marketSyncFailures[0]
	if failure.exchange != "okx" || failure.err == nil || failure.attemptedAt.IsZero() {
		t.Fatalf("unexpected market sync failure: %#v", failure)
	}
}

func TestMarketInstrumentRoutesRejectInvalidQuery(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	cases := []string{
		"/api/market/instruments",
		"/api/market/instruments?exchange=coinbase",
		"/api/market/instruments?exchange=binance&limit=zero",
		"/api/market/instruments?exchange=binance&limit=0",
		"/api/market/instruments?exchange=binance&status=delisted",
		"/api/market/candle-gaps",
		"/api/market/candle-gaps?exchange=binance&symbol=BTCUSDT&interval=tick",
		"/api/market/candle-gaps?exchange=binance&symbol=BTC-USDT&interval=1m",
		"/api/market/candle-gaps?exchange=binance&symbol=BTCUSDT&interval=1m&limit=101",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			recorder := serveAuthenticated(server, auth, http.MethodGet, path, "")
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			response := decodeAPIError(t, recorder)
			if response.Code != "invalid_request" {
				t.Fatalf("unexpected error response: %#v", response)
			}
		})
	}
}

type fakeInstrumentClient struct {
	instruments []data.MarketInstrument
	err         error
}

func (client fakeInstrumentClient) FetchInstruments(context.Context) ([]data.MarketInstrument, error) {
	if client.err != nil {
		return nil, client.err
	}
	return append([]data.MarketInstrument(nil), client.instruments...), nil
}

type mutableInstrumentClient struct {
	instruments []data.MarketInstrument
}

func (client *mutableInstrumentClient) FetchInstruments(context.Context) ([]data.MarketInstrument, error) {
	return append([]data.MarketInstrument(nil), client.instruments...), nil
}
