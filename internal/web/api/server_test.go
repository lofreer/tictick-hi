package api

import (
	"bytes"
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

	intentsRecorder := serveAuthenticated(
		server,
		cookie,
		http.MethodGet,
		"/api/backtests/"+created.ID+"/intents",
		"",
	)
	if intentsRecorder.Code != http.StatusOK {
		t.Fatalf("intents status = %d body = %s", intentsRecorder.Code, intentsRecorder.Body.String())
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

	liveExecuteBody := `{
		"name":"Live Execute EMA",
		"type":"live",
		"exchange":"binance",
		"accountId":"acct_live",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"execute","notificationChannel":"default"}
	}`
	liveExecuteRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", liveExecuteBody)
	if liveExecuteRecorder.Code != http.StatusBadRequest {
		t.Fatalf("live execute status = %d body = %s", liveExecuteRecorder.Code, liveExecuteRecorder.Body.String())
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
		"/api/trading/tasks/" + created.ID + "/executions",
		"/api/trading/tasks/" + created.ID + "/positions",
		"/api/trading/tasks/" + created.ID + "/notifications",
	} {
		recorder := serveAuthenticated(server, cookie, http.MethodGet, path, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSystemRoutes(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.notifications = append(repository.notifications, data.Notification{
		ID:           "nt_1",
		TaskID:       "tt_1",
		Channel:      "Ops",
		Provider:     "local",
		Target:       "ops",
		Title:        "Strategy intent",
		Body:         "signal",
		Status:       "failed",
		Error:        "demo failure",
		AttemptCount: 1,
		MaxAttempts:  3,
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	cases := []struct {
		path string
		body string
	}{
		{path: "/api/system/notifications/channels", body: `{"name":"Ops","provider":"webhook-demo","target":"demo://ops","enabled":true}`},
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

	notificationsRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/system/notifications", "")
	if notificationsRecorder.Code != http.StatusOK {
		t.Fatalf("notifications status = %d body = %s", notificationsRecorder.Code, notificationsRecorder.Body.String())
	}

	retryRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/system/notifications/nt_1/retry", "")
	if retryRecorder.Code != http.StatusOK {
		t.Fatalf("retry status = %d body = %s", retryRecorder.Code, retryRecorder.Body.String())
	}
}
