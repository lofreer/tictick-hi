package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestDataSyncTaskRoutes(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")

	createBody := bytes.NewBufferString(`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m"}`)
	createRecorder := httptest.NewRecorder()
	server.ServeHTTP(createRecorder, httptest.NewRequest(http.MethodPost, "/api/data/tasks", createBody))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}

	var created data.DataSyncTask
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Exchange != "binance" || created.Status != data.TaskStatusPending {
		t.Fatalf("unexpected created task: %#v", created)
	}

	startRecorder := httptest.NewRecorder()
	startPath := "/api/data/tasks/" + created.ID + "/realtime/start"
	server.ServeHTTP(startRecorder, httptest.NewRequest(http.MethodPost, startPath, nil))
	if startRecorder.Code != http.StatusOK {
		t.Fatalf("start status = %d body = %s", startRecorder.Code, startRecorder.Body.String())
	}

	listRecorder := httptest.NewRecorder()
	server.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/api/data/tasks", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}

	var tasks []data.DataSyncTask
	if err := json.NewDecoder(listRecorder.Body).Decode(&tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || !tasks[0].RealtimeEnabled {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}
}

func TestCandlesRoute(t *testing.T) {
	repository := newFakeRepository()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository.candles = append(repository.candles, data.Candle{
		Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
		OpenTime: now, CloseTime: now.Add(time.Minute),
		Open: "100.1", High: "101.2", Low: "99.9", Close: "100.8", Volume: "12.5",
		IsClosed: true,
	})

	server := NewServer(repository, "")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/candles?exchange=binance&symbol=BTCUSDT&interval=1m", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var candles []data.Candle
	if err := json.NewDecoder(recorder.Body).Decode(&candles); err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 || candles[0].Open != "100.1" {
		t.Fatalf("unexpected candles: %#v", candles)
	}
}

type fakeRepository struct {
	tasks   []data.DataSyncTask
	candles []data.Candle
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{}
}

func (repository *fakeRepository) ListDataSyncTasks(context.Context) ([]data.DataSyncTask, error) {
	return append([]data.DataSyncTask(nil), repository.tasks...), nil
}

func (repository *fakeRepository) CreateDataSyncTask(
	_ context.Context,
	request data.CreateDataSyncTask,
) (data.DataSyncTask, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task := data.DataSyncTask{
		ID:        "dst_1",
		Exchange:  request.Exchange,
		Symbol:    request.Symbol,
		Interval:  request.Interval,
		Status:    data.TaskStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.tasks = append(repository.tasks, task)
	return task, nil
}

func (repository *fakeRepository) DeleteDataSyncTask(_ context.Context, id string) error {
	for index, task := range repository.tasks {
		if task.ID == id {
			repository.tasks = append(repository.tasks[:index], repository.tasks[index+1:]...)
			return nil
		}
	}
	return data.ErrNotFound
}

func (repository *fakeRepository) SetSyncEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.DataSyncTask, error) {
	return repository.updateTask(ctx, id, func(task *data.DataSyncTask) {
		task.SyncEnabled = enabled
		task.Status = data.TaskStatusPending
		if !enabled {
			task.Status = data.TaskStatusPaused
		}
	})
}

func (repository *fakeRepository) SetRealtimeEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.DataSyncTask, error) {
	return repository.updateTask(ctx, id, func(task *data.DataSyncTask) {
		task.RealtimeEnabled = enabled
		task.Status = data.TaskStatusRunning
		if !enabled {
			task.Status = data.TaskStatusPaused
		}
	})
}

func (repository *fakeRepository) ListCandles(
	_ context.Context,
	query data.CandleQuery,
) ([]data.Candle, error) {
	var matches []data.Candle
	for _, candle := range repository.candles {
		if candle.Exchange == query.Exchange && candle.Symbol == query.Symbol && candle.Interval == query.Interval {
			matches = append(matches, candle)
		}
	}
	return matches, nil
}

func (repository *fakeRepository) updateTask(
	_ context.Context,
	id string,
	update func(task *data.DataSyncTask),
) (data.DataSyncTask, error) {
	for index := range repository.tasks {
		if repository.tasks[index].ID == id {
			update(&repository.tasks[index])
			return repository.tasks[index], nil
		}
	}
	return data.DataSyncTask{}, data.ErrNotFound
}
