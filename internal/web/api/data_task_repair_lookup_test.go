package api

import (
	"encoding/json"
	"errors"
	"net/http"
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
