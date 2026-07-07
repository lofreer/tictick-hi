package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestAuthSessionManagementRoutes(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")
	currentAuth := loginTestOperator(t, server)
	otherAuth := loginTestOperator(t, server)

	listRecorder := serveAuthenticated(server, currentAuth, http.MethodGet, "/api/auth/sessions", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list sessions status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var sessions []data.OperatorSession
	if err := json.NewDecoder(listRecorder.Body).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("session count = %d sessions = %#v", len(sessions), sessions)
	}
	currentID := ""
	otherID := ""
	for _, session := range sessions {
		if session.ID == "" {
			t.Fatalf("session id was empty: %#v", session)
		}
		if session.TokenHash != "" {
			t.Fatalf("session token hash leaked: %#v", session)
		}
		if session.Current {
			currentID = session.ID
		} else {
			otherID = session.ID
		}
	}
	if currentID == "" || otherID == "" {
		t.Fatalf("expected current and non-current sessions: %#v", sessions)
	}

	missingCSRFRecorder := serveAuthenticatedWithoutCSRF(
		server,
		currentAuth,
		http.MethodDelete,
		"/api/auth/sessions/"+otherID,
		"",
	)
	if missingCSRFRecorder.Code != http.StatusForbidden {
		t.Fatalf("missing csrf delete status = %d body = %s", missingCSRFRecorder.Code, missingCSRFRecorder.Body.String())
	}

	currentDeleteRecorder := serveAuthenticated(server, currentAuth, http.MethodDelete, "/api/auth/sessions/"+currentID, "")
	if currentDeleteRecorder.Code != http.StatusConflict {
		t.Fatalf("delete current session status = %d body = %s", currentDeleteRecorder.Code, currentDeleteRecorder.Body.String())
	}
	currentDeleteResponse := decodeAPIError(t, currentDeleteRecorder)
	if currentDeleteResponse.Code != "auth_current_session_revoke_forbidden" ||
		currentDeleteResponse.Message != "current session cannot be revoked" {
		t.Fatalf("unexpected delete current response: %#v", currentDeleteResponse)
	}
	failedRevokeEvent := assertAuditAction(t, repository.auditEvents, "auth.session_revoke", "operator_session", currentID)
	if failedRevokeEvent.Outcome != "failure" ||
		failedRevokeEvent.Metadata["reason"] != "current_session_revoke_forbidden" {
		t.Fatalf("unexpected current session revoke audit event: %#v", failedRevokeEvent)
	}

	deleteRecorder := serveAuthenticated(server, currentAuth, http.MethodDelete, "/api/auth/sessions/"+otherID, "")
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete other session status = %d body = %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}

	revokedRecorder := serveAuthenticated(server, otherAuth, http.MethodGet, "/api/auth/me", "")
	if revokedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status = %d body = %s", revokedRecorder.Code, revokedRecorder.Body.String())
	}

	remainingRecorder := serveAuthenticated(server, currentAuth, http.MethodGet, "/api/auth/sessions", "")
	if remainingRecorder.Code != http.StatusOK {
		t.Fatalf("remaining sessions status = %d body = %s", remainingRecorder.Code, remainingRecorder.Body.String())
	}
	sessions = nil
	if err := json.NewDecoder(remainingRecorder.Body).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || !sessions[0].Current || sessions[0].ID != currentID {
		t.Fatalf("unexpected remaining sessions: %#v", sessions)
	}
}

func TestLoginStoresSessionClientContext(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")

	body := bytes.NewBufferString(`{"username":"` + testUsername + `","password":"` + testPassword + `"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	request.RemoteAddr = "198.51.100.24:12345"
	request.Header.Set("X-Forwarded-For", "203.0.113.24, 198.51.100.24")
	request.Header.Set("User-Agent", "tictick-hi-test/1.0")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", recorder.Code, recorder.Body.String())
	}

	var auth authTestSession
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			auth.session = cookie
		}
		if cookie.Name == csrfCookieName {
			auth.csrf = cookie
		}
	}
	if auth.session == nil || auth.csrf == nil {
		t.Fatalf("missing auth cookies: %#v", recorder.Result().Cookies())
	}
	session := repository.sessions[sessionTokenHash(auth.session.Value)]
	if session.RemoteAddr != "203.0.113.24" || session.UserAgent != "tictick-hi-test/1.0" {
		t.Fatalf("unexpected stored session context: %#v", session)
	}

	listRecorder := serveAuthenticated(server, &auth, http.MethodGet, "/api/auth/sessions", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list sessions status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var sessions []data.OperatorSession
	if err := json.NewDecoder(listRecorder.Body).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 ||
		sessions[0].RemoteAddr != "203.0.113.24" ||
		sessions[0].UserAgent != "tictick-hi-test/1.0" {
		t.Fatalf("unexpected listed session context: %#v", sessions)
	}

	sameContextSessions := listSessionsWithClientContext(
		t,
		server,
		&auth,
		"198.51.100.24:12345",
		"203.0.113.24, 198.51.100.24",
		"tictick-hi-test/1.0",
	)
	if len(sameContextSessions) != 1 ||
		sameContextSessions[0].RemoteAddrChanged ||
		sameContextSessions[0].UserAgentChanged {
		t.Fatalf("same context was marked changed: %#v", sameContextSessions)
	}

	auditEventsBeforeContextChange := len(repository.auditEvents)
	changedContextSessions := listSessionsWithClientContext(
		t,
		server,
		&auth,
		"198.51.100.99:12345",
		"203.0.113.99, 198.51.100.99",
		"tictick-hi-test/2.0",
	)
	if len(changedContextSessions) != 1 ||
		!changedContextSessions[0].RemoteAddrChanged ||
		!changedContextSessions[0].UserAgentChanged {
		t.Fatalf("changed context was not marked changed: %#v", changedContextSessions)
	}
	event := assertAuditAction(
		t,
		repository.auditEvents[auditEventsBeforeContextChange:],
		"auth.session_context_change",
		"operator_session",
		changedContextSessions[0].ID,
	)
	if event.Outcome != "warning" ||
		event.Metadata["remoteAddrChanged"] != "true" ||
		event.Metadata["userAgentChanged"] != "true" {
		t.Fatalf("unexpected session context change audit event: %#v", event)
	}
}

func listSessionsWithClientContext(
	t *testing.T,
	server http.Handler,
	auth *authTestSession,
	remoteAddr string,
	forwardedFor string,
	userAgent string,
) []data.OperatorSession {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
	request.RemoteAddr = remoteAddr
	request.Header.Set("X-Forwarded-For", forwardedFor)
	request.Header.Set("User-Agent", userAgent)
	request.AddCookie(auth.session)
	request.AddCookie(auth.csrf)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list sessions status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var sessions []data.OperatorSession
	if err := json.NewDecoder(recorder.Body).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	return sessions
}
