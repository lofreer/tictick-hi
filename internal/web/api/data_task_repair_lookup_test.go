package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestDataSyncTaskSingleGapRepairUsesDirectTaskLookup(t *testing.T) {
	baseRepository := newFakeRepository()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	gapFrom := time.Date(2026, 6, 27, 3, 2, 0, 0, time.UTC)
	gapTo := time.Date(2026, 6, 27, 3, 4, 0, 0, time.UTC)
	baseRepository.tasks = []data.DataSyncTask{
		{
			ID:           "dst_direct_lookup",
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			Interval:     "1m",
			Status:       data.TaskStatusSucceeded,
			MarketStatus: data.DataSyncMarketStatusActive,
			DataHealth:   data.DataSyncHealthGap,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	baseRepository.taskGapDetails["dst_direct_lookup"] = data.DataSyncGapList{
		TaskID:      "dst_direct_lookup",
		Gaps:        []data.CandleGap{{From: gapFrom, To: gapTo, MissingCandles: 1}},
		TotalCount:  1,
		RepairLimit: 20,
	}
	repository := &failingListRepository{
		fakeRepository: baseRepository,
		err:            errors.New("list data sync tasks should not be used"),
	}
	server := NewServer(repository, "")
	cookie := loginTestOperator(t, server)

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks/dst_direct_lookup/repair-gap",
		`{"from":"`+gapFrom.Format(time.RFC3339)+`","to":"`+gapTo.Format(time.RFC3339)+`"}`,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("repair status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.DataSyncGapRepairResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.SourceTaskID != "dst_direct_lookup" || len(result.CreatedTasks) != 1 {
		t.Fatalf("unexpected direct lookup repair result: %#v", result)
	}
}

func TestDataSyncTaskDetailUsesDirectTaskLookup(t *testing.T) {
	baseRepository := newFakeRepository()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	baseRepository.tasks = []data.DataSyncTask{
		{
			ID:           "dst_detail_lookup",
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			Interval:     "1m",
			Status:       data.TaskStatusFailed,
			MarketStatus: data.DataSyncMarketStatusActive,
			DataHealth:   data.DataSyncHealthFailed,
			LastError:    `binance klines: Get "https://api.binance.com/api/v3/klines?symbol=BTCUSDT&limit=500": EOF`,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	repository := &failingListRepository{
		fakeRepository: baseRepository,
		err:            errors.New("list data sync tasks should not be used"),
	}
	server := NewServer(repository, "")
	cookie := loginTestOperator(t, server)

	recorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/data/tasks/dst_detail_lookup", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("detail status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var task data.DataSyncTask
	if err := json.NewDecoder(recorder.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.ID != "dst_detail_lookup" || task.Status != data.TaskStatusFailed {
		t.Fatalf("unexpected detail task: %#v", task)
	}
	if strings.Contains(task.LastError, "symbol=BTCUSDT") || !strings.Contains(task.LastError, "api.binance.com") {
		t.Fatalf("detail task last error was not sanitized: %q", task.LastError)
	}
}
