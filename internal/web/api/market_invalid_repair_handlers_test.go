package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestMarketCandleInvalidIssueRepairRouteQueuesSyncTasks(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	start := time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC)
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

	body := `{"exchange":"binance","symbol":"btcusdt","interval":"1m","openTimes":["2026-06-27T08:01:00Z","2026-06-27T08:02:00Z"]}`
	missingCSRF := serveAuthenticatedWithoutCSRF(server, auth, http.MethodPost, "/api/market/candle-invalid-issues/repair", body)
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status = %d body = %s", missingCSRF.Code, missingCSRF.Body.String())
	}

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/candle-invalid-issues/repair", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("repair status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.DataSyncGapRepairResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != "" || result.SkippedExisting != 0 || result.TotalCount != 2 ||
		result.RepairLimit != data.MaxMarketCandleInvalidIssueScanLimit {
		t.Fatalf("unexpected repair metadata: %#v", result)
	}
	if len(result.CreatedTasks) != 2 {
		t.Fatalf("created repair tasks = %#v, want two", result.CreatedTasks)
	}
	if result.CreatedTasks[0].StartTime == nil || !result.CreatedTasks[0].StartTime.Equal(start.Add(time.Minute)) ||
		result.CreatedTasks[0].EndTime == nil || !result.CreatedTasks[0].EndTime.Equal(start.Add(2*time.Minute)) ||
		result.CreatedTasks[1].StartTime == nil || !result.CreatedTasks[1].StartTime.Equal(start.Add(2*time.Minute)) ||
		result.CreatedTasks[1].EndTime == nil || !result.CreatedTasks[1].EndTime.Equal(start.Add(3*time.Minute)) {
		t.Fatalf("unexpected repair task windows: %#v", result.CreatedTasks)
	}

	duplicateRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/candle-invalid-issues/repair", body)
	if duplicateRecorder.Code != http.StatusOK {
		t.Fatalf("duplicate status = %d body = %s", duplicateRecorder.Code, duplicateRecorder.Body.String())
	}
	var duplicate data.DataSyncGapRepairResult
	if err := json.NewDecoder(duplicateRecorder.Body).Decode(&duplicate); err != nil {
		t.Fatal(err)
	}
	if len(duplicate.CreatedTasks) != 0 || duplicate.SkippedExisting != 2 {
		t.Fatalf("duplicate repair result = %#v, want skipped existing", duplicate)
	}

	notInvalid := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/market/candle-invalid-issues/repair",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m","openTimes":["2026-06-27T08:03:00Z"]}`,
	)
	if notInvalid.Code != http.StatusNotFound {
		t.Fatalf("not invalid status = %d body = %s", notInvalid.Code, notInvalid.Body.String())
	}
}

func TestMarketCandleInvalidIssueRepairRouteRequiresActiveMarketInstrument(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.marketInstruments = nil
	start := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	repository.candles = append(repository.candles, data.Candle{
		Exchange: "binance", Symbol: "ETHUSDT", Interval: "1m",
		OpenTime: start, CloseTime: start.Add(time.Minute),
		Open: "0", High: "101", Low: "99", Close: "100", Volume: "1",
		IsClosed: true,
	})

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/market/candle-invalid-issues/repair",
		`{"exchange":"binance","symbol":"ETHUSDT","interval":"1m","openTimes":["2026-06-27T09:00:00Z"]}`,
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
		t.Fatalf("repair created tasks for missing market: %#v", repository.tasks)
	}
}
