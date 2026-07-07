package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestSystemAuditEventsPageReturnsNextCursor(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)
	for _, username := range []string{"ops-page-a", "ops-page-b", "ops-page-c"} {
		recorder := serveAuthenticated(
			server,
			auth,
			http.MethodPost,
			"/api/system/operators",
			`{"username":"`+username+`","password":"secret123A","enabled":true}`,
		)
		if recorder.Code != http.StatusCreated {
			t.Fatalf("create operator status = %d body = %s", recorder.Code, recorder.Body.String())
		}
	}

	firstRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/audit-events/page?limit=2", "")
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("first page status = %d body = %s", firstRecorder.Code, firstRecorder.Body.String())
	}
	var firstPage data.AuditEventPage
	if err := json.NewDecoder(firstRecorder.Body).Decode(&firstPage); err != nil {
		t.Fatal(err)
	}
	if len(firstPage.Events) != 2 || firstPage.NextCursor == "" {
		t.Fatalf("first page = %#v, want two events and next cursor", firstPage)
	}

	secondRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/audit-events/page?limit=2&cursor="+url.QueryEscape(firstPage.NextCursor), "")
	if secondRecorder.Code != http.StatusOK {
		t.Fatalf("second page status = %d body = %s", secondRecorder.Code, secondRecorder.Body.String())
	}
	var secondPage data.AuditEventPage
	if err := json.NewDecoder(secondRecorder.Body).Decode(&secondPage); err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Events) != 2 {
		t.Fatalf("second page events = %#v, want two older events", secondPage.Events)
	}
	if firstPage.Events[0].ID == secondPage.Events[0].ID || firstPage.Events[1].ID == secondPage.Events[0].ID {
		t.Fatalf("second page overlapped first page: first=%#v second=%#v", firstPage.Events, secondPage.Events)
	}

	invalidRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/audit-events/page?cursor=bad", "")
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid cursor status = %d body = %s", invalidRecorder.Code, invalidRecorder.Body.String())
	}
}

func TestSystemAuditEventsExportReturnsCSV(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	createRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators",
		`{"username":"ops-export","password":"secret123A","enabled":true}`,
	)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create operator status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}

	exportRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/audit-events/export?limit=10", "")
	if exportRecorder.Code != http.StatusOK {
		t.Fatalf("export audit events status = %d body = %s", exportRecorder.Code, exportRecorder.Body.String())
	}
	if contentType := exportRecorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/csv") {
		t.Fatalf("content-type = %q, want text/csv", contentType)
	}
	if disposition := exportRecorder.Header().Get("Content-Disposition"); !strings.Contains(disposition, auditEventCSVFilename) {
		t.Fatalf("content-disposition = %q, want filename %q", disposition, auditEventCSVFilename)
	}
	body := exportRecorder.Body.String()
	if strings.Contains(body, "secret123A") {
		t.Fatalf("audit export leaked operator password: %s", body)
	}

	rows, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v\n%s", err, body)
	}
	if len(rows) < 2 {
		t.Fatalf("csv rows = %#v", rows)
	}
	if got := strings.Join(rows[0], ","); got != strings.Join(auditEventCSVHeader, ",") {
		t.Fatalf("csv header = %#v, want %#v", rows[0], auditEventCSVHeader)
	}
	createRow := auditCSVRowByAction(t, rows, "operator.create")
	columns := auditCSVColumns(rows[0])
	if createRow[columns["actorUsername"]] != testUsername {
		t.Fatalf("actor username column = %q, want %q", createRow[columns["actorUsername"]], testUsername)
	}
	var metadata map[string]string
	if err := json.Unmarshal([]byte(createRow[columns["metadata"]]), &metadata); err != nil {
		t.Fatalf("decode metadata column: %v", err)
	}
	if metadata["username"] != "ops-export" || metadata["enabled"] != "true" {
		t.Fatalf("metadata column = %#v", metadata)
	}
	if _, ok := metadata["password"]; ok {
		t.Fatalf("metadata leaked password key: %#v", metadata)
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

func TestAuditEventsCSVNeutralizesFormulaCells(t *testing.T) {
	payload, err := auditEventsCSV([]data.AuditEvent{
		{
			ID:           "ae_1",
			Action:       "=IMPORTXML()",
			ResourceType: "operator",
			ResourceID:   " +SUM(1,1)",
			Outcome:      "success",
			UserAgent:    "@evil",
			Metadata:     map[string]string{"note": "plain"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(payload))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("csv rows = %#v", rows)
	}
	columns := auditCSVColumns(rows[0])
	if rows[1][columns["action"]] != "'=IMPORTXML()" {
		t.Fatalf("action cell = %q", rows[1][columns["action"]])
	}
	if rows[1][columns["resourceId"]] != "' +SUM(1,1)" {
		t.Fatalf("resourceId cell = %q", rows[1][columns["resourceId"]])
	}
	if rows[1][columns["userAgent"]] != "'@evil" {
		t.Fatalf("userAgent cell = %q", rows[1][columns["userAgent"]])
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

func auditCSVColumns(header []string) map[string]int {
	columns := map[string]int{}
	for index, name := range header {
		columns[name] = index
	}
	return columns
}

func auditCSVRowByAction(t *testing.T, rows [][]string, action string) []string {
	t.Helper()
	columns := auditCSVColumns(rows[0])
	for _, row := range rows[1:] {
		if row[columns["action"]] == action {
			return row
		}
	}
	t.Fatalf("missing action %q in %#v", action, rows)
	return nil
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
