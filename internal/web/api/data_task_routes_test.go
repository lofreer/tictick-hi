package api

import (
	"encoding/json"
	"net/http"
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
}

func TestDataSyncTaskRoutesSanitizeLastError(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.tasks = []data.DataSyncTask{
		{
			ID:           "dst_legacy_error",
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			Interval:     "1m",
			Status:       data.TaskStatusPending,
			SyncEnabled:  true,
			LastError:    `binance klines: Get "https://api.binance.com/api/v3/klines?endTime=1782524388943&interval=1m&limit=500&startTime=1780277926000&symbol=BTCUSDT": EOF`,
			DataHealth:   data.DataSyncHealthSyncing,
			AttemptCount: 1,
			CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
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
	assertSanitizedTaskError(t, tasks[0].LastError)

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
	assertSanitizedTaskError(t, started.LastError)
}

func assertSanitizedTaskError(t *testing.T, value string) {
	t.Helper()
	if value == "" {
		t.Fatal("expected sanitized error")
	}
	for _, forbidden := range []string{"/api/v3/klines", "symbol=BTCUSDT", "endTime=", "startTime=", "https://"} {
		if strings.Contains(value, forbidden) {
			t.Fatalf("error leaks %q: %s", forbidden, value)
		}
	}
	if !strings.Contains(value, "api.binance.com") || !strings.Contains(value, "EOF") {
		t.Fatalf("sanitized error lost useful context: %s", value)
	}
}
