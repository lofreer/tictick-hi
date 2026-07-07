package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestSystemAuditEventsRouteRecordsSecurityActions(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)

	createRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators",
		`{"username":"ops-audit","password":"secret123A","enabled":true}`,
	)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create operator status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}

	disableRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/operators/op_ops-audit/disable", "")
	if disableRecorder.Code != http.StatusOK {
		t.Fatalf("disable operator status = %d body = %s", disableRecorder.Code, disableRecorder.Body.String())
	}

	listRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/audit-events?limit=10", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list audit events status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	body := listRecorder.Body.String()
	if strings.Contains(body, "secret123A") {
		t.Fatalf("audit response leaked operator password: %s", body)
	}

	var events []data.AuditEvent
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&events); err != nil {
		t.Fatal(err)
	}
	assertAuditAction(t, events, "auth.login", "operator", "op_admin")
	assertAuditAction(t, events, "operator.create", "operator", "op_ops-audit")
	assertAuditAction(t, events, "operator.disable", "operator", "op_ops-audit")

	if len(repository.auditEvents) < 3 {
		t.Fatalf("expected repository audit events, got %#v", repository.auditEvents)
	}
	for _, event := range events {
		if event.Action == "operator.create" && event.Metadata["username"] != "ops-audit" {
			t.Fatalf("unexpected operator create metadata: %#v", event.Metadata)
		}
		if event.ActorOperatorID == "" || event.ActorUsername != testUsername {
			t.Fatalf("unexpected actor on audit event: %#v", event)
		}
	}
}

func TestFailedLoginWritesAnonymousAuditEvent(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")

	recorder := httptestPostJSON(server, "/api/auth/login", `{"username":"admin","password":"wrong"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("login status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(repository.auditEvents) != 1 {
		t.Fatalf("audit events = %#v", repository.auditEvents)
	}
	event := repository.auditEvents[0]
	if event.Action != "auth.login" || event.Outcome != "failure" || event.ActorOperatorID != "" {
		t.Fatalf("unexpected login failure audit event: %#v", event)
	}
}

func TestAuditEventClientContextIsTrimmedAndBounded(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")
	userAgent := "  " + strings.Repeat("a", sessionContextMaxLength+10) + "  "

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		bytes.NewBufferString(`{"username":"admin","password":"wrong"}`),
	)
	request.RemoteAddr = "198.51.100.24:12345"
	request.Header.Set("X-Forwarded-For", " 203.0.113.24, 198.51.100.24")
	request.Header.Set("User-Agent", userAgent)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("login status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(repository.auditEvents) != 1 {
		t.Fatalf("audit events = %#v", repository.auditEvents)
	}
	event := repository.auditEvents[0]
	if event.RemoteAddr != "203.0.113.24" {
		t.Fatalf("audit remote addr = %q", event.RemoteAddr)
	}
	if len([]rune(event.UserAgent)) != sessionContextMaxLength {
		t.Fatalf("audit user agent length = %d, want %d", len([]rune(event.UserAgent)), sessionContextMaxLength)
	}
	if strings.Contains(event.UserAgent, " ") {
		t.Fatalf("audit user agent was not trimmed: %q", event.UserAgent)
	}
}

func TestParseAuditLimitNormalizesInvalidAndOversizedValues(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
	}{
		{name: "default", query: "", want: defaultAuditEventLimit},
		{name: "invalid", query: "?limit=abc", want: defaultAuditEventLimit},
		{name: "zero", query: "?limit=0", want: defaultAuditEventLimit},
		{name: "negative", query: "?limit=-5", want: defaultAuditEventLimit},
		{name: "valid", query: "?limit=25", want: 25},
		{name: "max", query: "?limit=500", want: maxAuditEventLimit},
		{name: "oversized", query: "?limit=501", want: maxAuditEventLimit},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/system/audit-events"+test.query, nil)
			if got := parseAuditLimit(request); got != test.want {
				t.Fatalf("parseAuditLimit() = %d, want %d", got, test.want)
			}
		})
	}
}

func assertAuditAction(t *testing.T, events []data.AuditEvent, action string, resourceType string, resourceID string) data.AuditEvent {
	t.Helper()
	for _, event := range events {
		if event.Action == action && event.ResourceType == resourceType && event.ResourceID == resourceID {
			return event
		}
	}
	t.Fatalf("missing audit action %s %s %s in %#v", action, resourceType, resourceID, events)
	return data.AuditEvent{}
}

func httptestPostJSON(server http.Handler, path string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}
