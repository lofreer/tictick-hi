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

func TestStrategiesRoute(t *testing.T) {
	server := NewServer(newFakeRepository(), "")

	listRecorder := httptest.NewRecorder()
	server.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/api/strategies", nil))

	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var strategies []map[string]any
	if err := json.NewDecoder(listRecorder.Body).Decode(&strategies); err != nil {
		t.Fatal(err)
	}
	if len(strategies) == 0 {
		t.Fatal("expected at least one strategy")
	}

	detailRecorder := httptest.NewRecorder()
	server.ServeHTTP(detailRecorder, httptest.NewRequest(http.MethodGet, "/api/strategies/ema-cross", nil))
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("detail status = %d body = %s", detailRecorder.Code, detailRecorder.Body.String())
	}
}

func TestBacktestRoutes(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")

	createBody := bytes.NewBufferString(`{
		"name":"EMA BTC backtest",
		"exchange":"binance",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-02T00:00:00Z",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"initialBalance":"10000",
		"feeBps":"1",
		"slippageBps":"0.5",
		"triggerMode":"closed_candle"
	}`)
	createRecorder := httptest.NewRecorder()
	server.ServeHTTP(createRecorder, httptest.NewRequest(http.MethodPost, "/api/backtests", createBody))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}

	var created data.BacktestTask
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Status != data.TaskStatusPending || created.StrategyID != "ema-cross" {
		t.Fatalf("unexpected created backtest: %#v", created)
	}

	listRecorder := httptest.NewRecorder()
	server.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/api/backtests", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}

	detailRecorder := httptest.NewRecorder()
	server.ServeHTTP(detailRecorder, httptest.NewRequest(http.MethodGet, "/api/backtests/"+created.ID, nil))
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("detail status = %d body = %s", detailRecorder.Code, detailRecorder.Body.String())
	}

	ordersRecorder := httptest.NewRecorder()
	server.ServeHTTP(ordersRecorder, httptest.NewRequest(http.MethodGet, "/api/backtests/"+created.ID+"/orders", nil))
	if ordersRecorder.Code != http.StatusOK {
		t.Fatalf("orders status = %d body = %s", ordersRecorder.Code, ordersRecorder.Body.String())
	}
}

func TestTradingTaskRoutes(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")

	createBody := bytes.NewBufferString(`{
		"name":"Paper EMA",
		"type":"paper",
		"exchange":"binance",
		"accountId":"paper",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"execute","notificationChannel":"default"}
	}`)
	createRecorder := httptest.NewRecorder()
	server.ServeHTTP(createRecorder, httptest.NewRequest(http.MethodPost, "/api/trading/tasks", createBody))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}

	var created data.TradingTask
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Type != "paper" || created.Status != data.TaskStatusPending {
		t.Fatalf("unexpected trading task: %#v", created)
	}

	liveBody := bytes.NewBufferString(`{
		"name":"Live EMA",
		"type":"live",
		"exchange":"binance",
		"accountId":"acct_live",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"notify","notificationChannel":"default"}
	}`)
	liveRecorder := httptest.NewRecorder()
	server.ServeHTTP(liveRecorder, httptest.NewRequest(http.MethodPost, "/api/trading/tasks", liveBody))
	if liveRecorder.Code != http.StatusCreated {
		t.Fatalf("live create status = %d body = %s", liveRecorder.Code, liveRecorder.Body.String())
	}

	startRecorder := httptest.NewRecorder()
	server.ServeHTTP(startRecorder, httptest.NewRequest(http.MethodPost, "/api/trading/tasks/"+created.ID+"/start", nil))
	if startRecorder.Code != http.StatusOK {
		t.Fatalf("start status = %d body = %s", startRecorder.Code, startRecorder.Body.String())
	}

	for _, path := range []string{
		"/api/trading/tasks",
		"/api/trading/tasks/" + created.ID,
		"/api/trading/tasks/" + created.ID + "/intents",
		"/api/trading/tasks/" + created.ID + "/orders",
		"/api/trading/tasks/" + created.ID + "/notifications",
	} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
		}
	}
}

type fakeRepository struct {
	backtestOrders map[string][]data.BacktestOrder
	backtests      []data.BacktestTask
	tradingTasks   []data.TradingTask
	tasks          []data.DataSyncTask
	candles        []data.Candle
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{backtestOrders: map[string][]data.BacktestOrder{}}
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

func (repository *fakeRepository) ListBacktestTasks(context.Context) ([]data.BacktestTask, error) {
	return append([]data.BacktestTask(nil), repository.backtests...), nil
}

func (repository *fakeRepository) CreateBacktestTask(
	_ context.Context,
	request data.CreateBacktestTask,
) (data.BacktestTask, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task := data.BacktestTask{
		ID:             "bt_1",
		Name:           request.Name,
		Exchange:       request.Exchange,
		Symbol:         request.Symbol,
		Interval:       request.Interval,
		StartTime:      request.StartTime,
		EndTime:        request.EndTime,
		StrategyID:     request.StrategyID,
		StrategyParams: request.StrategyParams,
		InitialBalance: request.InitialBalance,
		FeeBps:         request.FeeBps,
		SlippageBps:    request.SlippageBps,
		TriggerMode:    request.TriggerMode,
		Status:         data.TaskStatusPending,
		ResultSummary:  map[string]any{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repository.backtests = append(repository.backtests, task)
	return task, nil
}

func (repository *fakeRepository) GetBacktestTask(_ context.Context, id string) (data.BacktestTask, error) {
	for _, task := range repository.backtests {
		if task.ID == id {
			return task, nil
		}
	}
	return data.BacktestTask{}, data.ErrNotFound
}

func (repository *fakeRepository) ListBacktestOrders(
	_ context.Context,
	backtestID string,
) ([]data.BacktestOrder, error) {
	return append([]data.BacktestOrder(nil), repository.backtestOrders[backtestID]...), nil
}

func (repository *fakeRepository) ListTradingTasks(context.Context) ([]data.TradingTask, error) {
	return append([]data.TradingTask(nil), repository.tradingTasks...), nil
}

func (repository *fakeRepository) CreateTradingTask(
	_ context.Context,
	request data.CreateTradingTask,
) (data.TradingTask, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task := data.TradingTask{
		ID:             "tt_" + request.Type,
		Name:           request.Name,
		Type:           request.Type,
		Exchange:       request.Exchange,
		AccountID:      request.AccountID,
		Symbol:         request.Symbol,
		Interval:       request.Interval,
		StrategyID:     request.StrategyID,
		StrategyParams: request.StrategyParams,
		IntentPolicy:   request.IntentPolicy,
		Status:         data.TaskStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repository.tradingTasks = append(repository.tradingTasks, task)
	return task, nil
}

func (repository *fakeRepository) GetTradingTask(_ context.Context, id string) (data.TradingTask, error) {
	for _, task := range repository.tradingTasks {
		if task.ID == id {
			return task, nil
		}
	}
	return data.TradingTask{}, data.ErrNotFound
}

func (repository *fakeRepository) SetTradingTaskStatus(
	_ context.Context,
	id string,
	status data.TaskStatus,
) (data.TradingTask, error) {
	for index := range repository.tradingTasks {
		if repository.tradingTasks[index].ID == id {
			repository.tradingTasks[index].Status = status
			return repository.tradingTasks[index], nil
		}
	}
	return data.TradingTask{}, data.ErrNotFound
}

func (repository *fakeRepository) ListTradingIntents(context.Context, string) ([]data.StrategyIntent, error) {
	return nil, nil
}

func (repository *fakeRepository) ListTradingOrders(context.Context, string) ([]data.Order, error) {
	return nil, nil
}

func (repository *fakeRepository) ListTradingNotifications(context.Context, string) ([]data.Notification, error) {
	return nil, nil
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
