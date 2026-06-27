package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

const (
	testUsername = "admin"
	testPassword = "secret123"
)

func newAuthenticatedTestServer(t *testing.T) (*fakeRepository, http.Handler, *http.Cookie) {
	t.Helper()

	repository := newFakeRepository()
	server := NewServer(repository, "")
	cookie := loginTestOperator(t, server)
	return repository, server, cookie
}

func loginTestOperator(t *testing.T, server http.Handler) *http.Cookie {
	t.Helper()

	body := bytes.NewBufferString(`{"username":"` + testUsername + `","password":"` + testPassword + `"}`)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			return cookie
		}
	}
	t.Fatal("login did not set session cookie")
	return nil
}

func serveAuthenticated(
	server http.Handler,
	cookie *http.Cookie,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func TestAPIRequiresAuthentication(t *testing.T) {
	server := NewServer(newFakeRepository(), "")

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/strategies", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAuthRoutes(t *testing.T) {
	server := NewServer(newFakeRepository(), "")
	cookie := loginTestOperator(t, server)

	meRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/auth/me", "")
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("me status = %d body = %s", meRecorder.Code, meRecorder.Body.String())
	}

	logoutRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/auth/logout", "")
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("logout status = %d body = %s", logoutRecorder.Code, logoutRecorder.Body.String())
	}

	afterLogout := serveAuthenticated(server, cookie, http.MethodGet, "/api/auth/me", "")
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("after logout status = %d body = %s", afterLogout.Code, afterLogout.Body.String())
	}
}

func TestServeFrontendSupportsGetAndHead(t *testing.T) {
	staticRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticRoot, "index.html"), []byte("<!doctype html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := NewServer(newFakeRepository(), staticRoot)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(method, "/", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", method, recorder.Code, recorder.Body.String())
		}
	}
}

func TestDataSyncTaskRoutes(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

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
	if created.Exchange != "binance" || created.Status != data.TaskStatusPending {
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
	if len(tasks) != 1 || !tasks[0].RealtimeEnabled {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}
}

func TestCandlesRoute(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository.candles = append(repository.candles, data.Candle{
		Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m",
		OpenTime: now, CloseTime: now.Add(time.Minute),
		Open: "100.1", High: "101.2", Low: "99.9", Close: "100.8", Volume: "12.5",
		IsClosed: true,
	})

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/candles?exchange=binance&symbol=BTCUSDT&interval=1m",
		"",
	)

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
	_, server, cookie := newAuthenticatedTestServer(t)

	listRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/strategies", "")

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

	detailRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/strategies/ema-cross", "")
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("detail status = %d body = %s", detailRecorder.Code, detailRecorder.Body.String())
	}
}

func TestBacktestRoutes(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	createBody := `{
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
	}`
	createRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/backtests", createBody)
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

	listRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/backtests", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}

	detailRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/backtests/"+created.ID, "")
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("detail status = %d body = %s", detailRecorder.Code, detailRecorder.Body.String())
	}

	ordersRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/backtests/"+created.ID+"/orders",
		"",
	)
	if ordersRecorder.Code != http.StatusOK {
		t.Fatalf("orders status = %d body = %s", ordersRecorder.Code, ordersRecorder.Body.String())
	}
}

func TestTradingTaskRoutes(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	createBody := `{
		"name":"Paper EMA",
		"type":"paper",
		"exchange":"binance",
		"accountId":"paper",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"execute","notificationChannel":"default"}
	}`
	createRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", createBody)
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

	liveBody := `{
		"name":"Live EMA",
		"type":"live",
		"exchange":"binance",
		"accountId":"acct_live",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"notify","notificationChannel":"default"}
	}`
	liveRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", liveBody)
	if liveRecorder.Code != http.StatusCreated {
		t.Fatalf("live create status = %d body = %s", liveRecorder.Code, liveRecorder.Body.String())
	}

	startRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/trading/tasks/"+created.ID+"/start",
		"",
	)
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
		recorder := serveAuthenticated(server, cookie, http.MethodGet, path, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSystemRoutes(t *testing.T) {
	_, server, cookie := newAuthenticatedTestServer(t)

	cases := []struct {
		path string
		body string
	}{
		{path: "/api/system/notifications/channels", body: `{"name":"Ops","provider":"webhook","target":"https://example.test","enabled":true}`},
		{path: "/api/system/exchange-accounts", body: `{"exchange":"binance","alias":"main","apiKey":"key","apiSecret":"secret","enabled":true}`},
		{path: "/api/system/operators", body: `{"username":"ops","password":"secret123","enabled":true}`},
	}
	for _, item := range cases {
		createRecorder := serveAuthenticated(server, cookie, http.MethodPost, item.path, item.body)
		if createRecorder.Code != http.StatusCreated {
			t.Fatalf("%s create status = %d body = %s", item.path, createRecorder.Code, createRecorder.Body.String())
		}

		listRecorder := serveAuthenticated(server, cookie, http.MethodGet, item.path, "")
		if listRecorder.Code != http.StatusOK {
			t.Fatalf("%s list status = %d body = %s", item.path, listRecorder.Code, listRecorder.Body.String())
		}
	}

	healthRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/system/health", "")
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("health status = %d body = %s", healthRecorder.Code, healthRecorder.Body.String())
	}
}

type fakeRepository struct {
	backtestOrders map[string][]data.BacktestOrder
	backtests      []data.BacktestTask
	channels       []data.NotificationChannel
	accounts       []data.ExchangeAccount
	operators      []data.Operator
	passwords      map[string]string
	sessions       map[string]data.OperatorSession
	tradingTasks   []data.TradingTask
	tasks          []data.DataSyncTask
	candles        []data.Candle
}

func newFakeRepository() *fakeRepository {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		backtestOrders: map[string][]data.BacktestOrder{},
		passwords:      map[string]string{},
		sessions:       map[string]data.OperatorSession{},
	}
	operator := data.Operator{
		ID:        "op_admin",
		Username:  testUsername,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.operators = append(repository.operators, operator)
	repository.passwords[operator.ID] = testPassword
	return repository
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

func (repository *fakeRepository) ListNotificationChannels(context.Context) ([]data.NotificationChannel, error) {
	return append([]data.NotificationChannel(nil), repository.channels...), nil
}

func (repository *fakeRepository) CreateNotificationChannel(
	_ context.Context,
	request data.CreateNotificationChannel,
) (data.NotificationChannel, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	channel := data.NotificationChannel{
		ID:        "nc_1",
		Name:      request.Name,
		Provider:  request.Provider,
		Target:    request.Target,
		Enabled:   request.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.channels = append(repository.channels, channel)
	return channel, nil
}

func (repository *fakeRepository) ListExchangeAccounts(context.Context) ([]data.ExchangeAccount, error) {
	return append([]data.ExchangeAccount(nil), repository.accounts...), nil
}

func (repository *fakeRepository) CreateExchangeAccount(
	_ context.Context,
	request data.CreateExchangeAccount,
) (data.ExchangeAccount, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	account := data.ExchangeAccount{
		ID:        "ea_1",
		Exchange:  request.Exchange,
		Alias:     request.Alias,
		Enabled:   request.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.accounts = append(repository.accounts, account)
	return account, nil
}

func (repository *fakeRepository) ListOperators(context.Context) ([]data.Operator, error) {
	return append([]data.Operator(nil), repository.operators...), nil
}

func (repository *fakeRepository) CreateOperator(
	_ context.Context,
	request data.CreateOperator,
) (data.Operator, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	operator := data.Operator{
		ID:        "op_" + request.Username,
		Username:  request.Username,
		Enabled:   request.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.operators = append(repository.operators, operator)
	repository.passwords[operator.ID] = request.Password
	return operator, nil
}

func (repository *fakeRepository) AuthenticateOperator(
	_ context.Context,
	username string,
	password string,
) (data.Operator, error) {
	for _, operator := range repository.operators {
		if operator.Username == username && operator.Enabled && repository.passwords[operator.ID] == password {
			return operator, nil
		}
	}
	return data.Operator{}, data.ErrUnauthorized
}

func (repository *fakeRepository) CreateOperatorSession(
	_ context.Context,
	session data.OperatorSession,
) error {
	repository.sessions[session.TokenHash] = session
	return nil
}

func (repository *fakeRepository) GetOperatorBySession(
	_ context.Context,
	tokenHash string,
	now time.Time,
) (data.Operator, error) {
	session, exists := repository.sessions[tokenHash]
	if !exists || !session.ExpiresAt.After(now) {
		return data.Operator{}, data.ErrUnauthorized
	}
	for _, operator := range repository.operators {
		if operator.ID == session.OperatorID && operator.Enabled {
			return operator, nil
		}
	}
	return data.Operator{}, data.ErrUnauthorized
}

func (repository *fakeRepository) DeleteOperatorSession(_ context.Context, tokenHash string) error {
	delete(repository.sessions, tokenHash)
	return nil
}

func (repository *fakeRepository) SystemHealth(context.Context) (data.SystemHealth, error) {
	return data.SystemHealth{
		Status:    "ok",
		Database:  "ok",
		CheckedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Services:  []data.ServiceHealth{{Name: "api", Status: "ok"}},
	}, nil
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
