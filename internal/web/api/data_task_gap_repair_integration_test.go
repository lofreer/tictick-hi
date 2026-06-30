package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationDataSyncTaskGapRepairRouteConvergesSourceHealth(t *testing.T) {
	store, pool, ctx := openAPIIntegrationStore(t)
	server := NewServer(store, "")

	symbol := apiIntegrationSymbol("APIDG")
	username := fmt.Sprintf("api-data-gap-%d", time.Now().UTC().UnixNano())
	password := "secret123"
	start := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	end := start.Add(5 * time.Minute)
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

	markAPIIntegrationDataSyncTaskRunning(t, ctx, pool, source.ID, "api-data-gap-source-worker")
	lastOpenTime := start.Add(5 * time.Minute)
	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:   source.ID,
		WorkerID: "api-data-gap-source-worker",
		Candles: []data.Candle{
			apiIntegrationCandle(symbol, start, 0),
			apiIntegrationCandle(symbol, start, 1),
			apiIntegrationCandle(symbol, start, 5),
		},
		LastOpenTime: &lastOpenTime,
		Completed:    true,
	}); err != nil {
		t.Fatal(err)
	}

	before := getAPIIntegrationDataSyncTask(t, server, auth, source.ID)
	if before.DataHealth != data.DataSyncHealthGap || before.GapSummary == nil ||
		before.GapSummary.Count != 1 || before.GapSummary.FirstGap == nil ||
		!before.GapSummary.FirstGap.From.Equal(start.Add(2*time.Minute)) ||
		!before.GapSummary.FirstGap.To.Equal(end) ||
		before.GapSummary.FirstGap.MissingCandles != 3 {
		t.Fatalf("source task before repair = %#v, want one task-window gap", before)
	}

	gapsRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/data/tasks/"+source.ID+"/gaps", "")
	if gapsRecorder.Code != http.StatusOK {
		t.Fatalf("gaps status = %d body = %s", gapsRecorder.Code, gapsRecorder.Body.String())
	}
	var gaps data.DataSyncGapList
	if err := json.NewDecoder(gapsRecorder.Body).Decode(&gaps); err != nil {
		t.Fatal(err)
	}
	gapFrom := start.Add(2 * time.Minute)
	gapTo := end
	if gaps.TaskID != source.ID || gaps.TotalCount != 1 || gaps.ReturnedCount != 1 ||
		gaps.RepairLimit != 20 || len(gaps.Gaps) != 1 ||
		!gaps.Gaps[0].From.Equal(gapFrom) ||
		!gaps.Gaps[0].To.Equal(gapTo) ||
		gaps.Gaps[0].MissingCandles != 3 {
		t.Fatalf("gaps before repair = %#v, want exact task-window gap", gaps)
	}

	repairBody := fmt.Sprintf(`{"from":%q,"to":%q}`, gapFrom.Format(time.RFC3339), gapTo.Format(time.RFC3339))
	repairRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/data/tasks/"+source.ID+"/repair-gap", repairBody)
	if repairRecorder.Code != http.StatusOK {
		t.Fatalf("repair gap status = %d body = %s", repairRecorder.Code, repairRecorder.Body.String())
	}
	var repair data.DataSyncGapRepairResult
	if err := json.NewDecoder(repairRecorder.Body).Decode(&repair); err != nil {
		t.Fatal(err)
	}
	if repair.SourceTaskID != source.ID || repair.TotalCount != 1 || repair.RepairLimit != 1 ||
		repair.SkippedExisting != 0 || len(repair.CreatedTasks) != 1 ||
		repair.CreatedTasks[0].RepairSourceTaskID != source.ID ||
		repair.CreatedTasks[0].StartTime == nil ||
		!repair.CreatedTasks[0].StartTime.Equal(gapFrom) ||
		repair.CreatedTasks[0].EndTime == nil ||
		!repair.CreatedTasks[0].EndTime.Equal(gapTo) ||
		!repair.CreatedTasks[0].SyncEnabled ||
		repair.CreatedTasks[0].RealtimeEnabled {
		t.Fatalf("repair result = %#v, want one source-linked repair task", repair)
	}
	repairTask := repair.CreatedTasks[0]

	markAPIIntegrationDataSyncTaskRunning(t, ctx, pool, repairTask.ID, "api-data-gap-repair-worker")
	lastRepairOpenTime := start.Add(4 * time.Minute)
	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:   repairTask.ID,
		WorkerID: "api-data-gap-repair-worker",
		Candles: []data.Candle{
			apiIntegrationCandle(symbol, start, 2),
			apiIntegrationCandle(symbol, start, 3),
			apiIntegrationCandle(symbol, start, 4),
		},
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
	if after.DataHealth != data.DataSyncHealthOK || after.GapSummary != nil {
		t.Fatalf("source task after repair = %#v, want healthy source window", after)
	}

	afterGapsRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/data/tasks/"+source.ID+"/gaps", "")
	if afterGapsRecorder.Code != http.StatusOK {
		t.Fatalf("gaps after status = %d body = %s", afterGapsRecorder.Code, afterGapsRecorder.Body.String())
	}
	var afterGaps data.DataSyncGapList
	if err := json.NewDecoder(afterGapsRecorder.Body).Decode(&afterGaps); err != nil {
		t.Fatal(err)
	}
	if afterGaps.TotalCount != 0 || afterGaps.ReturnedCount != 0 || len(afterGaps.Gaps) != 0 || afterGaps.Limited {
		t.Fatalf("gaps after repair = %#v, want no source task gaps", afterGaps)
	}
}

func markAPIIntegrationDataSyncTaskRunning(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	taskID string,
	workerID string,
) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET status = $2,
		       locked_by = $3,
		       locked_until = now() + interval '1 minute',
		       heartbeat_at = now()
		 WHERE id = $1`,
		taskID,
		data.TaskStatusRunning,
		workerID,
	); err != nil {
		t.Fatal(err)
	}
}

func getAPIIntegrationDataSyncTask(
	t *testing.T,
	server http.Handler,
	auth *authTestSession,
	taskID string,
) data.DataSyncTask {
	t.Helper()

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/data/tasks", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("list data sync tasks status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var tasks []data.DataSyncTask
	if err := json.NewDecoder(recorder.Body).Decode(&tasks); err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if task.ID == taskID {
			return task
		}
	}
	t.Fatalf("data sync task %s not found in API list: %#v", taskID, tasks)
	return data.DataSyncTask{}
}
