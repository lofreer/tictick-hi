package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationMarketCandleGapRepairRouteConvergesPostgresScan(t *testing.T) {
	store, pool, ctx := openAPIIntegrationStore(t)
	server := NewServer(store, "")

	symbol := apiIntegrationSymbol("APIGC")
	username := fmt.Sprintf("api-gap-%d", time.Now().UTC().UnixNano())
	password := "secret123A"
	start := time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC)
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
	insertAPIIntegrationCandle(t, ctx, pool, apiIntegrationCandle(symbol, start, 0))
	insertAPIIntegrationCandle(t, ctx, pool, apiIntegrationCandle(symbol, start, 1))
	insertAPIIntegrationCandle(t, ctx, pool, apiIntegrationCandle(symbol, start, 3))

	scanPath := "/api/market/candle-gaps?exchange=binance&symbol=" +
		url.QueryEscape(symbol) + "&interval=1m"
	beforeRecorder := serveAuthenticated(server, auth, http.MethodGet, scanPath, "")
	if beforeRecorder.Code != http.StatusOK {
		t.Fatalf("scan before status = %d body = %s", beforeRecorder.Code, beforeRecorder.Body.String())
	}
	var before data.MarketCandleGapScan
	if err := json.NewDecoder(beforeRecorder.Body).Decode(&before); err != nil {
		t.Fatal(err)
	}
	gapFrom := start.Add(2 * time.Minute)
	gapTo := start.Add(3 * time.Minute)
	if before.Window.Count != 3 || before.TotalCount != 1 || before.ReturnedCount != 1 ||
		len(before.Gaps) != 1 || before.Gaps[0].MissingCandles != 1 ||
		!before.Gaps[0].From.Equal(gapFrom) || !before.Gaps[0].To.Equal(gapTo) {
		t.Fatalf("gap scan before repair = %#v, want one persisted gap", before)
	}

	body := fmt.Sprintf(
		`{"exchange":"binance","symbol":%q,"interval":"1m","from":%q,"to":%q}`,
		strings.ToLower(symbol),
		gapFrom.Format(time.RFC3339),
		gapTo.Format(time.RFC3339),
	)
	repairRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/candle-gaps/repair", body)
	if repairRecorder.Code != http.StatusOK {
		t.Fatalf("repair status = %d body = %s", repairRecorder.Code, repairRecorder.Body.String())
	}
	var repair data.DataSyncGapRepairResult
	if err := json.NewDecoder(repairRecorder.Body).Decode(&repair); err != nil {
		t.Fatal(err)
	}
	if repair.SourceTaskID != "" || repair.TotalCount != 1 || repair.RepairLimit != 1 ||
		repair.SkippedExisting != 0 || len(repair.CreatedTasks) != 1 ||
		repair.CreatedTasks[0].RepairSourceTaskID != "" ||
		repair.CreatedTasks[0].StartTime == nil ||
		!repair.CreatedTasks[0].StartTime.Equal(gapFrom) ||
		repair.CreatedTasks[0].EndTime == nil ||
		!repair.CreatedTasks[0].EndTime.Equal(gapTo) {
		t.Fatalf("unexpected gap repair result: %#v", repair)
	}
	repairTask := repair.CreatedTasks[0]

	if _, err := pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET status = $2,
		       locked_by = 'api-gap-repair-worker',
		       locked_until = now() + interval '1 minute',
		       heartbeat_at = now()
		 WHERE id = $1`,
		repairTask.ID,
		data.TaskStatusRunning,
	); err != nil {
		t.Fatal(err)
	}

	repairedCandle := apiIntegrationCandle(symbol, start, 2)
	lastOpenTime := repairedCandle.OpenTime
	if err := store.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:       repairTask.ID,
		WorkerID:     "api-gap-repair-worker",
		Candles:      []data.Candle{repairedCandle},
		LastOpenTime: &lastOpenTime,
		Completed:    true,
	}); err != nil {
		t.Fatal(err)
	}

	afterRecorder := serveAuthenticated(server, auth, http.MethodGet, scanPath, "")
	if afterRecorder.Code != http.StatusOK {
		t.Fatalf("scan after status = %d body = %s", afterRecorder.Code, afterRecorder.Body.String())
	}
	var after data.MarketCandleGapScan
	if err := json.NewDecoder(afterRecorder.Body).Decode(&after); err != nil {
		t.Fatal(err)
	}
	if after.Window.Count != 4 || after.TotalCount != 0 || after.ReturnedCount != 0 ||
		len(after.Gaps) != 0 || after.Limited {
		t.Fatalf("gap scan after repair result = %#v, want contiguous history", after)
	}
}
