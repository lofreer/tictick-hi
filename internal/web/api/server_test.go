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

type authTestSession struct {
	session *http.Cookie
	csrf    *http.Cookie
}

func newAuthenticatedTestServer(t *testing.T) (*fakeRepository, http.Handler, *authTestSession) {
	t.Helper()

	repository := newFakeRepository()
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)
	return repository, server, auth
}

func loginTestOperator(t *testing.T, server http.Handler) *authTestSession {
	t.Helper()

	body := bytes.NewBufferString(`{"username":"` + testUsername + `","password":"` + testPassword + `"}`)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	auth := &authTestSession{}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			auth.session = cookie
		}
		if cookie.Name == csrfCookieName {
			auth.csrf = cookie
		}
	}
	if auth.session == nil {
		t.Fatal("login did not set session cookie")
	}
	if auth.csrf == nil {
		t.Fatal("login did not set csrf cookie")
	}
	return auth
}

func serveAuthenticated(
	server http.Handler,
	auth *authTestSession,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.AddCookie(auth.session)
	request.AddCookie(auth.csrf)
	if !isSafeMethod(method) {
		request.Header.Set(csrfHeaderName, auth.csrf.Value)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func serveAuthenticatedWithoutCSRF(
	server http.Handler,
	auth *authTestSession,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.AddCookie(auth.session)
	request.AddCookie(auth.csrf)
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
	auth := loginTestOperator(t, server)

	meRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/auth/me", "")
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("me status = %d body = %s", meRecorder.Code, meRecorder.Body.String())
	}

	logoutRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/auth/logout", "")
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("logout status = %d body = %s", logoutRecorder.Code, logoutRecorder.Body.String())
	}

	afterLogout := serveAuthenticated(server, auth, http.MethodGet, "/api/auth/me", "")
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("after logout status = %d body = %s", afterLogout.Code, afterLogout.Body.String())
	}
}

func TestCSRFProtectionRejectsUnsafeRequestsWithoutToken(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticatedWithoutCSRF(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m"}`,
	)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginRateLimit(t *testing.T) {
	repository := newFakeRepository()
	server := NewServerWithConfig(repository, Config{
		LoginFailureLimit:  2,
		LoginFailureWindow: time.Minute,
		LoginLockout:       time.Hour,
	})

	body := `{"username":"` + testUsername + `","password":"wrong"}`
	for index := 0; index < 2; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
		request.RemoteAddr = "203.0.113.10:12345"
		server.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d body = %s", index+1, recorder.Code, recorder.Body.String())
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	request.RemoteAddr = "203.0.113.10:12345"
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("limited status = %d body = %s", recorder.Code, recorder.Body.String())
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

	repository.tasks[0].Status = data.TaskStatusFailed
	repository.tasks[0].SyncEnabled = false
	repository.tasks[0].RealtimeEnabled = false
	repository.tasks[0].LastError = "invalid symbol"

	retryPath := "/api/data/tasks/" + created.ID + "/retry"
	retryRecorder := serveAuthenticated(server, cookie, http.MethodPost, retryPath, "")
	if retryRecorder.Code != http.StatusOK {
		t.Fatalf("retry status = %d body = %s", retryRecorder.Code, retryRecorder.Body.String())
	}
	var retried data.DataSyncTask
	if err := json.NewDecoder(retryRecorder.Body).Decode(&retried); err != nil {
		t.Fatal(err)
	}
	if retried.Status != data.TaskStatusPending || !retried.SyncEnabled || retried.LastError != "" {
		t.Fatalf("unexpected retried task: %#v", retried)
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
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.accounts = append(repository.accounts,
		data.ExchangeAccount{
			ID:               "acct_live",
			Exchange:         "binance",
			Alias:            "main",
			Enabled:          true,
			CredentialStatus: "encrypted",
		},
		data.ExchangeAccount{
			ID:               "acct_disabled",
			Exchange:         "binance",
			Alias:            "disabled",
			Enabled:          false,
			CredentialStatus: "encrypted",
		},
		data.ExchangeAccount{
			ID:               "acct_legacy",
			Exchange:         "binance",
			Alias:            "legacy",
			Enabled:          true,
			CredentialStatus: "legacy",
		},
	)

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

	disabledLiveBody := `{
		"name":"Disabled Live EMA",
		"type":"live",
		"exchange":"binance",
		"accountId":"acct_disabled",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"notify","notificationChannel":"default"}
	}`
	disabledLiveRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", disabledLiveBody)
	if disabledLiveRecorder.Code != http.StatusBadRequest {
		t.Fatalf("disabled live status = %d body = %s", disabledLiveRecorder.Code, disabledLiveRecorder.Body.String())
	}

	legacyLiveBody := `{
		"name":"Legacy Live EMA",
		"type":"live",
		"exchange":"binance",
		"accountId":"acct_legacy",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"notify","notificationChannel":"default"}
	}`
	legacyLiveRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", legacyLiveBody)
	if legacyLiveRecorder.Code != http.StatusBadRequest {
		t.Fatalf("legacy live status = %d body = %s", legacyLiveRecorder.Code, legacyLiveRecorder.Body.String())
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
		{path: "/api/system/notifications/channels", body: `{"name":"Ops","provider":"webhook","target":"https://example.invalid/ops","enabled":true}`},
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

	disableOperatorRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/system/operators/op_ops/disable", "")
	if disableOperatorRecorder.Code != http.StatusOK {
		t.Fatalf("disable operator status = %d body = %s", disableOperatorRecorder.Code, disableOperatorRecorder.Body.String())
	}
	var disabledOperator data.Operator
	if err := json.NewDecoder(disableOperatorRecorder.Body).Decode(&disabledOperator); err != nil {
		t.Fatal(err)
	}
	if disabledOperator.Enabled {
		t.Fatalf("operator was not disabled: %#v", disabledOperator)
	}

	healthRecorder := serveAuthenticated(server, cookie, http.MethodGet, "/api/system/health", "")
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("health status = %d body = %s", healthRecorder.Code, healthRecorder.Body.String())
	}
	var health data.SystemHealth
	if err := json.NewDecoder(healthRecorder.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if len(health.Services) < 2 || health.Services[1].LastHeartbeatAt == nil || health.Services[1].LockedUntil == nil {
		t.Fatalf("health response does not expose worker lease fields: %#v", health.Services)
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
