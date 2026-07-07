package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

const (
	testUsername = "admin"
	testPassword = "secret123A"
)

type authTestSession struct {
	session *http.Cookie
	csrf    *http.Cookie
}

type testAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
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
	return loginOperator(t, server, testUsername, testPassword)
}

func loginOperator(t *testing.T, server http.Handler, username string, password string) *authTestSession {
	t.Helper()

	body := bytes.NewBufferString(`{"username":"` + username + `","password":"` + password + `"}`)
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

func decodeAPIError(t *testing.T, recorder *httptest.ResponseRecorder) testAPIError {
	t.Helper()

	var response testAPIError
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != response.Message {
		t.Fatalf("legacy error field must match message: %#v", response)
	}
	return response
}

func TestAPIRequiresAuthentication(t *testing.T) {
	server := NewServer(newFakeRepository(), "")

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/strategies", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "unauthorized" || response.Message != "unauthorized" {
		t.Fatalf("unexpected auth error response: %#v", response)
	}
}

func TestServerAssignsRequestID(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/health", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	requestID := recorder.Header().Get(requestIDHeaderName)
	if !isValidRequestID(requestID) {
		t.Fatalf("invalid response request id: %q", requestID)
	}
	if repository.lastSystemHealthRequestID != requestID {
		t.Fatalf("context request id = %q, want %q", repository.lastSystemHealthRequestID, requestID)
	}
}

func TestServerReusesValidRequestID(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/api/system/health", nil)
	request.Header.Set(requestIDHeaderName, "request-id-123")
	request.AddCookie(auth.session)
	request.AddCookie(auth.csrf)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get(requestIDHeaderName); got != "request-id-123" {
		t.Fatalf("response request id = %q", got)
	}
	if repository.lastSystemHealthRequestID != "request-id-123" {
		t.Fatalf("context request id = %q", repository.lastSystemHealthRequestID)
	}
}

func TestServerReplacesInvalidRequestIDWithoutEchoingValue(t *testing.T) {
	server := NewServer(newFakeRepository(), "")
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	request.Header.Set(requestIDHeaderName, "stage8_config_secret!")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	requestID := recorder.Header().Get(requestIDHeaderName)
	if !isValidRequestID(requestID) {
		t.Fatalf("invalid generated request id: %q", requestID)
	}
	if strings.Contains(requestID, "stage8_config_secret") {
		t.Fatalf("response leaked invalid request id: %q", requestID)
	}
}

func TestServerAccessLogIncludesRequestIDWithoutQuery(t *testing.T) {
	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})
	server := NewServer(newFakeRepository(), "")
	request := httptest.NewRequest(http.MethodGet, "/readyz?token=stage8_config_secret", nil)
	request.Header.Set(requestIDHeaderName, "request-id-123")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	logged := output.String()
	for _, expected := range []string{
		`"msg":"http request"`,
		`"request_id":"request-id-123"`,
		`"method":"GET"`,
		`"path":"/readyz"`,
		`"status":200`,
	} {
		if !strings.Contains(logged, expected) {
			t.Fatalf("access log missing %s: %s", expected, logged)
		}
	}
	if strings.Contains(logged, "stage8_config_secret") || strings.Contains(logged, "token=") {
		t.Fatalf("access log leaked query string: %s", logged)
	}
}

func TestAPIStructuredErrorResponses(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	csrfRecorder := serveAuthenticatedWithoutCSRF(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m"}`,
	)
	if csrfRecorder.Code != http.StatusForbidden {
		t.Fatalf("csrf status = %d body = %s", csrfRecorder.Code, csrfRecorder.Body.String())
	}
	csrfResponse := decodeAPIError(t, csrfRecorder)
	if csrfResponse.Code != "csrf_invalid" || csrfResponse.Message != "csrf token is invalid" {
		t.Fatalf("unexpected csrf error response: %#v", csrfResponse)
	}

	invalidRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m","extra":true}`,
	)
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d body = %s", invalidRecorder.Code, invalidRecorder.Body.String())
	}
	invalidResponse := decodeAPIError(t, invalidRecorder)
	if invalidResponse.Code != "invalid_request" || invalidResponse.Message == "" {
		t.Fatalf("unexpected invalid request response: %#v", invalidResponse)
	}

	repository, conflictServer, conflictAuth := newAuthenticatedTestServer(t)
	createRecorder := serveAuthenticated(
		conflictServer,
		conflictAuth,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"binance","symbol":"BTCUSDT","interval":"1m"}`,
	)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}
	conflictRecorder := serveAuthenticated(conflictServer, conflictAuth, http.MethodPost, "/api/data/tasks/"+repository.tasks[0].ID+"/retry", "")
	if conflictRecorder.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d body = %s", conflictRecorder.Code, conflictRecorder.Body.String())
	}
	conflictResponse := decodeAPIError(t, conflictRecorder)
	if conflictResponse.Code != "data_sync_retry_requires_failed" ||
		conflictResponse.Message != "data sync task must be failed before retry" {
		t.Fatalf("unexpected conflict response: %#v", conflictResponse)
	}
}

func TestAPIStructuredInternalErrorDoesNotLeakDetails(t *testing.T) {
	repository := &failingListRepository{
		fakeRepository: newFakeRepository(),
		err:            errors.New("database password leaked in driver detail"),
	}
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/data/tasks", "")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	response := decodeAPIError(t, recorder)
	if response.Code != "internal_error" || response.Message != "internal server error" {
		t.Fatalf("unexpected internal error response: %#v", response)
	}
	if strings.Contains(body, "password") {
		t.Fatalf("internal response leaked details: %s", body)
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
		{path: "/api/system/operators", body: `{"username":"ops","password":"secret123A","enabled":true}`},
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
