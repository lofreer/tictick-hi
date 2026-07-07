package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationDataSyncTaskInvalidRepairRouteConvergesSourceHealth(t *testing.T) {
	store, pool, ctx := openAPIIntegrationStore(t)
	server := NewServer(store, "")

	symbol := apiIntegrationSymbol("APIDI")
	username := fmt.Sprintf("api-data-invalid-%d", time.Now().UTC().UnixNano())
	password := "secret123A"
	start := time.Date(2026, 6, 27, 13, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Minute)
	invalidOpenTime := start.Add(2 * time.Minute)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		cleanupAPIIntegrationMarket(t, cleanupCtx, pool, symbol, username)
	})

	if _, _, err := store.EnsureOperator(ctx, data.CreateOperator{
		Username: username,
		Password: password,
		Enabled:  true,
	}); err != nil {
		t.Fatal(err)
	}
	auth := loginIntegrationOperator(t, server, username, password)
	upsertAPIIntegrationMarketInstrument(t, ctx, pool, symbol)

	createBody := fmt.Sprintf(
		`{"exchange":"binance","symbol":%q,"interval":"1m","startTime":%q,"endTime":%q}`,
		symbol,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)
	createRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/data/tasks", createBody)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create data sync task status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}
	var source data.DataSyncTask
	if err := json.NewDecoder(createRecorder.Body).Decode(&source); err != nil {
		t.Fatal(err)
	}
	if source.Exchange != "binance" || source.Symbol != symbol || source.Interval != "1m" ||
		source.StartTime == nil || !source.StartTime.Equal(start) ||
		source.EndTime == nil || !source.EndTime.Equal(end) {
		t.Fatalf("created source task = %#v", source)
	}

	markAPIIntegrationDataSyncTaskRunning(t, ctx, pool, source.ID, "api-data-invalid-source-worker")
	lastOpenTime := end
	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:   source.ID,
		WorkerID: "api-data-invalid-source-worker",
		Candles: []data.Candle{
			apiIntegrationCandle(symbol, start, 0),
			apiIntegrationCandle(symbol, start, 1),
			apiIntegrationCandle(symbol, start, 3),
		},
		LastOpenTime: &lastOpenTime,
		Completed:    true,
	}); err != nil {
		t.Fatal(err)
	}
	insertAPIIntegrationInvalidCandle(t, ctx, pool, symbol, invalidOpenTime)

	before := getAPIIntegrationDataSyncTask(t, server, auth, source.ID)
	if before.DataHealth != data.DataSyncHealthInvalid || before.InvalidSummary == nil ||
		before.InvalidSummary.Count != 1 || before.InvalidSummary.FirstIssue == nil ||
		before.InvalidSummary.FirstIssue.Code != data.CandleIssueInvalidOpenPrice ||
		before.InvalidSummary.FirstIssue.OpenTime == nil ||
		!before.InvalidSummary.FirstIssue.OpenTime.Equal(invalidOpenTime) ||
		before.GapSummary != nil {
		t.Fatalf("source task before repair = %#v, want one invalid task-window issue and no gap", before)
	}

	issuesPath := "/api/data/tasks/" + source.ID +
		"/invalid-issues?code=" + data.CandleIssueInvalidOpenPrice +
		"&from=" + invalidOpenTime.Format(time.RFC3339) +
		"&to=" + invalidOpenTime.Format(time.RFC3339)
	issuesRecorder := serveAuthenticated(server, auth, http.MethodGet, issuesPath, "")
	if issuesRecorder.Code != http.StatusOK {
		t.Fatalf("invalid issues status = %d body = %s", issuesRecorder.Code, issuesRecorder.Body.String())
	}
	var issues data.DataSyncInvalidIssueList
	if err := json.NewDecoder(issuesRecorder.Body).Decode(&issues); err != nil {
		t.Fatal(err)
	}
	if issues.TaskID != source.ID || issues.TotalCount != 1 || issues.ReturnedCount != 1 ||
		issues.IssueLimit != data.DefaultDataSyncInvalidIssueLimit || issues.Offset != 0 ||
		len(issues.Issues) != 1 ||
		issues.Issues[0].Code != data.CandleIssueInvalidOpenPrice ||
		issues.Issues[0].OpenTime == nil ||
		!issues.Issues[0].OpenTime.Equal(invalidOpenTime) {
		t.Fatalf("invalid issues before repair = %#v, want exact task-window invalid issue", issues)
	}

	repairBody := fmt.Sprintf(
		`{"code":%q,"from":%q,"to":%q}`,
		data.CandleIssueInvalidOpenPrice,
		invalidOpenTime.Format(time.RFC3339),
		invalidOpenTime.Format(time.RFC3339),
	)
	missingCSRF := serveAuthenticatedWithoutCSRF(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks/"+source.ID+"/repair-invalid-issues",
		repairBody,
	)
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing csrf repair status = %d body = %s", missingCSRF.Code, missingCSRF.Body.String())
	}

	repairRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks/"+source.ID+"/repair-invalid-issues",
		repairBody,
	)
	if repairRecorder.Code != http.StatusOK {
		t.Fatalf("repair invalid status = %d body = %s", repairRecorder.Code, repairRecorder.Body.String())
	}
	var repair data.DataSyncGapRepairResult
	if err := json.NewDecoder(repairRecorder.Body).Decode(&repair); err != nil {
		t.Fatal(err)
	}
	if repair.SourceTaskID != source.ID || repair.TotalCount != 1 || repair.RepairLimit != 20 ||
		repair.SkippedExisting != 0 || len(repair.CreatedTasks) != 1 ||
		repair.CreatedTasks[0].RepairSourceTaskID != source.ID ||
		repair.CreatedTasks[0].StartTime == nil ||
		!repair.CreatedTasks[0].StartTime.Equal(invalidOpenTime) ||
		repair.CreatedTasks[0].EndTime == nil ||
		!repair.CreatedTasks[0].EndTime.Equal(invalidOpenTime.Add(time.Minute)) ||
		!repair.CreatedTasks[0].SyncEnabled ||
		repair.CreatedTasks[0].RealtimeEnabled {
		t.Fatalf("repair invalid result = %#v, want one source-linked repair task", repair)
	}
	repairTask := repair.CreatedTasks[0]

	markAPIIntegrationDataSyncTaskRunning(t, ctx, pool, repairTask.ID, "api-data-invalid-repair-worker")
	lastRepairOpenTime := invalidOpenTime
	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:       repairTask.ID,
		WorkerID:     "api-data-invalid-repair-worker",
		Candles:      []data.Candle{apiIntegrationCandle(symbol, start, 2)},
		LastOpenTime: &lastRepairOpenTime,
		Completed:    true,
	}); err != nil {
		t.Fatal(err)
	}

	repaired := getAPIIntegrationDataSyncTask(t, server, auth, repairTask.ID)
	if repaired.Status != data.TaskStatusSucceeded || repaired.SyncEnabled ||
		repaired.LatestSyncedOpenTime == nil ||
		!repaired.LatestSyncedOpenTime.Equal(lastRepairOpenTime) {
		t.Fatalf("repair task after worker writeback = %#v", repaired)
	}

	after := getAPIIntegrationDataSyncTask(t, server, auth, source.ID)
	if after.DataHealth != data.DataSyncHealthOK || after.InvalidSummary != nil || after.GapSummary != nil {
		t.Fatalf("source task after invalid repair = %#v, want healthy source window", after)
	}

	afterIssuesRecorder := serveAuthenticated(server, auth, http.MethodGet, issuesPath, "")
	if afterIssuesRecorder.Code != http.StatusOK {
		t.Fatalf("invalid issues after status = %d body = %s", afterIssuesRecorder.Code, afterIssuesRecorder.Body.String())
	}
	var afterIssues data.DataSyncInvalidIssueList
	if err := json.NewDecoder(afterIssuesRecorder.Body).Decode(&afterIssues); err != nil {
		t.Fatal(err)
	}
	if afterIssues.TotalCount != 0 || afterIssues.ReturnedCount != 0 || len(afterIssues.Issues) != 0 || afterIssues.Limited {
		t.Fatalf("invalid issues after repair = %#v, want no source task invalid issues", afterIssues)
	}
}
