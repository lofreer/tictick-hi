package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestDataSyncTaskRoutes(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)

	createRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m"}`,
	)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}

	var created data.DataSyncTask
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Exchange != "binance" ||
		created.Status != data.TaskStatusPending ||
		created.DataHealth != data.DataSyncHealthInsufficient {
		t.Fatalf("unexpected created task: %#v", created)
	}

	startPath := "/api/data/tasks/" + created.ID + "/realtime/start"
	startRecorder := serveAuthenticated(server, cookie, http.MethodPost, startPath, "")
	if startRecorder.Code != http.StatusOK {
		t.Fatalf("start status = %d body = %s", startRecorder.Code, startRecorder.Body.String())
	}

	listRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/data/tasks", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}

	var tasks []data.DataSyncTask
	if err := json.NewDecoder(listRecorder.Body).Decode(&tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 ||
		!tasks[0].RealtimeEnabled ||
		tasks[0].DataHealth != data.DataSyncHealthSyncing {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}

	invalidRetryRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/"+created.ID+"/retry",
		"",
	)
	if invalidRetryRecorder.Code != http.StatusConflict {
		t.Fatalf("invalid retry status = %d body = %s", invalidRetryRecorder.Code, invalidRetryRecorder.Body.String())
	}
	invalidRetryResponse := decodeAPIError(t, invalidRetryRecorder)
	if invalidRetryResponse.Code != "data_sync_retry_requires_failed" {
		t.Fatalf("unexpected invalid retry response: %#v", invalidRetryResponse)
	}

	repository.tasks[0].Status = data.TaskStatusFailed
	repository.tasks[0].SyncEnabled = false
	repository.tasks[0].RealtimeEnabled = false
	repository.tasks[0].LastError = "invalid symbol"

	invalidCommandRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/"+created.ID+"/sync/start",
		"",
	)
	if invalidCommandRecorder.Code != http.StatusConflict {
		t.Fatalf("invalid command status = %d body = %s", invalidCommandRecorder.Code, invalidCommandRecorder.Body.String())
	}
	invalidCommandResponse := decodeAPIError(t, invalidCommandRecorder)
	if invalidCommandResponse.Code != "data_sync_command_invalid_state" {
		t.Fatalf("unexpected invalid command response: %#v", invalidCommandResponse)
	}

	retryPath := "/api/data/tasks/" + created.ID + "/retry"
	retryRecorder := serveAuthenticated(server, cookie, http.MethodPost, retryPath, "")
	if retryRecorder.Code != http.StatusOK {
		t.Fatalf("retry status = %d body = %s", retryRecorder.Code, retryRecorder.Body.String())
	}
	var retried data.DataSyncTask
	if err := json.NewDecoder(retryRecorder.Body).Decode(&retried); err != nil {
		t.Fatal(err)
	}
	if retried.Status != data.TaskStatusPending ||
		!retried.SyncEnabled ||
		retried.LastError != "" ||
		retried.DataHealth != data.DataSyncHealthSyncing {
		t.Fatalf("unexpected retried task: %#v", retried)
	}

	gapFrom := time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC)
	gapTo := time.Date(2026, 1, 1, 0, 3, 0, 0, time.UTC)
	repository.tasks[0].GapSummary = &data.DataSyncGapSummary{
		Count: 2,
		FirstGap: &data.CandleGap{
			From:           gapFrom,
			To:             gapTo,
			MissingCandles: 1,
		},
	}
	repository.taskGapDetails[created.ID] = data.DataSyncGapList{
		TaskID: created.ID,
		Gaps: []data.CandleGap{
			{From: gapFrom, To: gapTo, MissingCandles: 1},
			{From: gapTo.Add(time.Minute), To: gapTo.Add(3 * time.Minute), MissingCandles: 2},
		},
		Limited:       false,
		TotalCount:    2,
		ReturnedCount: 2,
		RepairLimit:   20,
	}

	gapsRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/data/tasks/"+created.ID+"/gaps",
		"",
	)
	if gapsRecorder.Code != http.StatusOK {
		t.Fatalf("gaps status = %d body = %s", gapsRecorder.Code, gapsRecorder.Body.String())
	}
	var gapList data.DataSyncGapList
	if err := json.NewDecoder(gapsRecorder.Body).Decode(&gapList); err != nil {
		t.Fatal(err)
	}
	if gapList.TaskID != created.ID ||
		gapList.Limited ||
		gapList.TotalCount != 2 ||
		gapList.ReturnedCount != 2 ||
		gapList.RepairLimit != 20 ||
		len(gapList.Gaps) != 2 ||
		!gapList.Gaps[0].From.Equal(gapFrom) ||
		gapList.Gaps[1].MissingCandles != 2 {
		t.Fatalf("unexpected gap list: %#v", gapList)
	}

	invalidOpenTime := gapFrom.Add(5 * time.Minute)
	secondInvalidOpenTime := invalidOpenTime.Add(time.Minute)
	repository.tasks[0].InvalidSummary = &data.DataSyncInvalidSummary{
		Count: 2,
		FirstIssue: &data.CandleIssue{
			Code:     "invalid_open_price",
			Message:  "open price value must be positive",
			OpenTime: &invalidOpenTime,
		},
	}
	repository.taskInvalidDetails[created.ID] = data.DataSyncInvalidIssueList{
		TaskID: created.ID,
		Issues: []data.CandleIssue{
			{Code: "invalid_open_price", Message: "open price value must be positive", OpenTime: &invalidOpenTime},
			{Code: "invalid_close_price", Message: "close price value must be positive", OpenTime: &secondInvalidOpenTime},
		},
		TotalCount: 2,
	}

	invalidRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/data/tasks/"+created.ID+"/invalid-issues?limit=1&offset=1",
		"",
	)
	if invalidRecorder.Code != http.StatusOK {
		t.Fatalf("invalid issues status = %d body = %s", invalidRecorder.Code, invalidRecorder.Body.String())
	}
	var invalidList data.DataSyncInvalidIssueList
	if err := json.NewDecoder(invalidRecorder.Body).Decode(&invalidList); err != nil {
		t.Fatal(err)
	}
	if invalidList.TaskID != created.ID ||
		invalidList.Limited ||
		invalidList.TotalCount != 2 ||
		invalidList.ReturnedCount != 1 ||
		invalidList.IssueLimit != 1 ||
		invalidList.Offset != 1 ||
		len(invalidList.Issues) != 1 ||
		invalidList.Issues[0].Code != "invalid_close_price" ||
		invalidList.Issues[0].OpenTime == nil ||
		!invalidList.Issues[0].OpenTime.Equal(secondInvalidOpenTime) {
		t.Fatalf("unexpected invalid issue list: %#v", invalidList)
	}

	filteredInvalidRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/data/tasks/"+created.ID+"/invalid-issues?code=invalid_close_price&from="+url.QueryEscape(secondInvalidOpenTime.Format(time.RFC3339))+"&to="+url.QueryEscape(secondInvalidOpenTime.Format(time.RFC3339)),
		"",
	)
	if filteredInvalidRecorder.Code != http.StatusOK {
		t.Fatalf("filtered invalid issues status = %d body = %s", filteredInvalidRecorder.Code, filteredInvalidRecorder.Body.String())
	}
	var filteredInvalidList data.DataSyncInvalidIssueList
	if err := json.NewDecoder(filteredInvalidRecorder.Body).Decode(&filteredInvalidList); err != nil {
		t.Fatal(err)
	}
	if filteredInvalidList.TaskID != created.ID ||
		filteredInvalidList.Limited ||
		filteredInvalidList.TotalCount != 1 ||
		filteredInvalidList.ReturnedCount != 1 ||
		filteredInvalidList.Issues[0].Code != "invalid_close_price" ||
		filteredInvalidList.Issues[0].OpenTime == nil ||
		!filteredInvalidList.Issues[0].OpenTime.Equal(secondInvalidOpenTime) {
		t.Fatalf("unexpected filtered invalid issue list: %#v", filteredInvalidList)
	}

	invalidBadQueryRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/data/tasks/"+created.ID+"/invalid-issues?offset=-1",
		"",
	)
	if invalidBadQueryRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid bad query status = %d body = %s", invalidBadQueryRecorder.Code, invalidBadQueryRecorder.Body.String())
	}
	invalidBadCodeRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/data/tasks/"+created.ID+"/invalid-issues?code=unknown",
		"",
	)
	if invalidBadCodeRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid bad code status = %d body = %s", invalidBadCodeRecorder.Code, invalidBadCodeRecorder.Body.String())
	}

	repairRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/"+created.ID+"/repair-gaps",
		"",
	)
	if repairRecorder.Code != http.StatusOK {
		t.Fatalf("repair status = %d body = %s", repairRecorder.Code, repairRecorder.Body.String())
	}
	var repairResult data.DataSyncGapRepairResult
	if err := json.NewDecoder(repairRecorder.Body).Decode(&repairResult); err != nil {
		t.Fatal(err)
	}
	if repairResult.SourceTaskID != created.ID ||
		len(repairResult.CreatedTasks) != 1 ||
		repairResult.TotalCount != 2 ||
		repairResult.RepairLimit != 20 ||
		repairResult.CreatedTasks[0].StartTime == nil ||
		!repairResult.CreatedTasks[0].StartTime.Equal(gapFrom) ||
		repairResult.CreatedTasks[0].RepairSourceTaskID != created.ID ||
		!repairResult.CreatedTasks[0].SyncEnabled {
		t.Fatalf("unexpected repair result: %#v", repairResult)
	}

	singleGapFrom := gapTo.Add(time.Minute)
	singleGapTo := gapTo.Add(3 * time.Minute)
	repairOneRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/"+created.ID+"/repair-gap",
		`{"from":"`+singleGapFrom.Format(time.RFC3339)+`","to":"`+singleGapTo.Format(time.RFC3339)+`"}`,
	)
	if repairOneRecorder.Code != http.StatusOK {
		t.Fatalf("repair one status = %d body = %s", repairOneRecorder.Code, repairOneRecorder.Body.String())
	}
	var repairOneResult data.DataSyncGapRepairResult
	if err := json.NewDecoder(repairOneRecorder.Body).Decode(&repairOneResult); err != nil {
		t.Fatal(err)
	}
	if repairOneResult.SourceTaskID != created.ID ||
		len(repairOneResult.CreatedTasks) != 1 ||
		repairOneResult.TotalCount != 1 ||
		repairOneResult.RepairLimit != 1 ||
		repairOneResult.CreatedTasks[0].StartTime == nil ||
		!repairOneResult.CreatedTasks[0].StartTime.Equal(singleGapFrom) ||
		repairOneResult.CreatedTasks[0].EndTime == nil ||
		!repairOneResult.CreatedTasks[0].EndTime.Equal(singleGapTo) ||
		repairOneResult.CreatedTasks[0].RepairSourceTaskID != created.ID ||
		!repairOneResult.CreatedTasks[0].SyncEnabled {
		t.Fatalf("unexpected single repair result: %#v", repairOneResult)
	}

	duplicateOneRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/"+created.ID+"/repair-gap",
		`{"from":"`+singleGapFrom.Format(time.RFC3339)+`","to":"`+singleGapTo.Format(time.RFC3339)+`"}`,
	)
	if duplicateOneRecorder.Code != http.StatusOK {
		t.Fatalf("duplicate repair one status = %d body = %s", duplicateOneRecorder.Code, duplicateOneRecorder.Body.String())
	}
	var duplicateOneResult data.DataSyncGapRepairResult
	if err := json.NewDecoder(duplicateOneRecorder.Body).Decode(&duplicateOneResult); err != nil {
		t.Fatal(err)
	}
	if len(duplicateOneResult.CreatedTasks) != 0 || duplicateOneResult.SkippedExisting != 1 {
		t.Fatalf("unexpected duplicate single repair result: %#v", duplicateOneResult)
	}

	nonGapFrom := gapFrom.Add(30 * time.Minute)
	nonGapTo := gapFrom.Add(31 * time.Minute)
	nonGapRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/"+created.ID+"/repair-gap",
		`{"from":"`+nonGapFrom.Format(time.RFC3339)+`","to":"`+nonGapTo.Format(time.RFC3339)+`"}`,
	)
	if nonGapRecorder.Code != http.StatusNotFound {
		t.Fatalf("non-gap repair one status = %d body = %s", nonGapRecorder.Code, nonGapRecorder.Body.String())
	}
	nonGapResponse := decodeAPIError(t, nonGapRecorder)
	if nonGapResponse.Code != "not_found" {
		t.Fatalf("unexpected non-gap repair response: %#v", nonGapResponse)
	}

	invalidOneRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/"+created.ID+"/repair-gap",
		`{"from":"`+singleGapTo.Format(time.RFC3339)+`","to":"`+singleGapFrom.Format(time.RFC3339)+`"}`,
	)
	if invalidOneRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid repair one status = %d body = %s", invalidOneRecorder.Code, invalidOneRecorder.Body.String())
	}
}

func TestDataSyncTaskDeleteRouteHidesTask(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.tasks = []data.DataSyncTask{
		{
			ID:           "dst_delete",
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			Interval:     "1m",
			Status:       data.TaskStatusRunning,
			SyncEnabled:  true,
			MarketStatus: data.DataSyncMarketStatusActive,
			DataHealth:   data.DataSyncHealthSyncing,
			CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	deleteRecorder := serveAuthenticated(server, cookie, http.MethodDelete, "/api/data/tasks/dst_delete", "")
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body = %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
	listRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/data/tasks", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var tasks []data.DataSyncTask
	if err := json.NewDecoder(listRecorder.Body).Decode(&tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Fatalf("tasks after delete = %#v, want empty list", tasks)
	}
	secondDeleteRecorder := serveAuthenticated(server, cookie, http.MethodDelete, "/api/data/tasks/dst_delete", "")
	if secondDeleteRecorder.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d body = %s", secondDeleteRecorder.Code, secondDeleteRecorder.Body.String())
	}
}

func TestDataSyncTaskCommandRejectsInactiveMarketInstrument(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.tasks = []data.DataSyncTask{
		{
			ID:           "dst_inactive",
			Exchange:     "binance",
			Symbol:       "SOLUSDT",
			Interval:     "1m",
			Status:       data.TaskStatusPaused,
			MarketStatus: data.DataSyncMarketStatusInactive,
			DataHealth:   data.DataSyncHealthPaused,
			CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/dst_inactive/sync/start",
		"",
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "market_instrument_not_active" ||
		response.Message != "market instrument is not active in catalog" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if repository.tasks[0].SyncEnabled {
		t.Fatalf("inactive task should not be started: %#v", repository.tasks[0])
	}
}

func TestDataSyncTaskRoutesSanitizeLastError(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.tasks = []data.DataSyncTask{
		{
			ID:                   "dst_legacy_error",
			Exchange:             "binance",
			Symbol:               "BTCUSDT",
			Interval:             "1m",
			Status:               data.TaskStatusPending,
			SyncEnabled:          true,
			LastError:            `binance klines: Get "https://api.binance.com/api/v3/klines?endTime=1782524388943&interval=1m&limit=500&startTime=1780277926000&symbol=BTCUSDT": EOF`,
			ExchangeBackoffError: `binance klines temporary unavailable: Get "https://api.binance.com/api/v3/klines?symbol=BTCUSDT": EOF`,
			DataHealth:           data.DataSyncHealthSyncing,
			AttemptCount:         1,
			CreatedAt:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	listRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/data/tasks", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var tasks []data.DataSyncTask
	if err := json.NewDecoder(listRecorder.Body).Decode(&tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks length = %d, want 1", len(tasks))
	}
	assertSanitizedTaskError(t, tasks[0].LastError, "binance klines: api.binance.com: EOF")
	assertSanitizedTaskError(t, tasks[0].ExchangeBackoffError, "binance klines temporary unavailable: api.binance.com: EOF")

	startRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/dst_legacy_error/realtime/start",
		"",
	)
	if startRecorder.Code != http.StatusOK {
		t.Fatalf("start status = %d body = %s", startRecorder.Code, startRecorder.Body.String())
	}
	var started data.DataSyncTask
	if err := json.NewDecoder(startRecorder.Body).Decode(&started); err != nil {
		t.Fatal(err)
	}
	assertSanitizedTaskError(t, started.LastError, "binance klines: api.binance.com: EOF")
	assertSanitizedTaskError(t, started.ExchangeBackoffError, "binance klines temporary unavailable: api.binance.com: EOF")
}

func assertSanitizedTaskError(t *testing.T, value string, expected string) {
	t.Helper()
	if value != expected {
		t.Fatalf("sanitized error = %q, want %q", value, expected)
	}
	for _, forbidden := range []string{`Get "`, "/api/v3/klines", "symbol=BTCUSDT", "endTime=", "startTime=", "https://"} {
		if strings.Contains(value, forbidden) {
			t.Fatalf("error leaks %q: %s", forbidden, value)
		}
	}
}
