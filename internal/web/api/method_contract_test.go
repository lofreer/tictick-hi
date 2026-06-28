package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIMethodNotAllowedContracts(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	authLoginRecorder := httptest.NewRecorder()
	server.ServeHTTP(authLoginRecorder, httptest.NewRequest(http.MethodGet, "/api/auth/login", nil))
	assertMethodNotAllowed(t, authLoginRecorder, http.MethodPost)

	cases := []struct {
		method string
		path   string
		allow  string
	}{
		{method: http.MethodPost, path: "/api/candles", allow: http.MethodGet},
		{method: http.MethodPut, path: "/api/data/tasks", allow: http.MethodGet + ", " + http.MethodPost},
		{method: http.MethodGet, path: "/api/data/tasks/dst_1/retry", allow: http.MethodPost},
		{method: http.MethodPost, path: "/api/data/tasks/dst_1/gaps", allow: http.MethodGet},
		{method: http.MethodGet, path: "/api/data/tasks/dst_1/repair-gaps", allow: http.MethodPost},
		{method: http.MethodGet, path: "/api/data/tasks/dst_1/repair-gap", allow: http.MethodPost},
		{method: http.MethodPost, path: "/api/backtests/bt_1/orders", allow: http.MethodGet},
		{method: http.MethodGet, path: "/api/trading/tasks/tt_1/start", allow: http.MethodPost},
		{method: http.MethodPost, path: "/api/trading/tasks/tt_1/orders", allow: http.MethodGet},
		{method: http.MethodPost, path: "/api/system/health", allow: http.MethodGet},
		{method: http.MethodPost, path: "/api/system/api-contract", allow: http.MethodGet},
		{method: http.MethodGet, path: "/api/system/notifications/nt_1/retry", allow: http.MethodPost},
		{method: http.MethodGet, path: "/api/system/operators/op_admin/disable", allow: http.MethodPost},
	}
	for _, item := range cases {
		t.Run(item.method+" "+item.path, func(t *testing.T) {
			recorder := serveAuthenticated(server, auth, item.method, item.path, "")
			assertMethodNotAllowed(t, recorder, item.allow)
		})
	}

	notFoundRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/not-a-route", "")
	if notFoundRecorder.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d body = %s", notFoundRecorder.Code, notFoundRecorder.Body.String())
	}
	if allow := notFoundRecorder.Header().Get("Allow"); allow != "" {
		t.Fatalf("unknown route must not set Allow header, got %q", allow)
	}
}

func assertMethodNotAllowed(t *testing.T, recorder *httptest.ResponseRecorder, allow string) {
	t.Helper()

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Allow"); got != allow {
		t.Fatalf("Allow header = %q, want %q", got, allow)
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "method_not_allowed" || response.Message != "method not allowed" {
		t.Fatalf("unexpected method error response: %#v", response)
	}
}
