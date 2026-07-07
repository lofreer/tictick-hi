package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestAdminListsOperatorSessions(t *testing.T) {
	_, server, adminAuth := newAuthenticatedTestServer(t)
	operator := createOperatorForSessionTests(t, server, adminAuth)
	loginOperator(t, server, operator.Username, "secret123A")
	loginOperator(t, server, operator.Username, "secret123A")

	recorder := serveAuthenticated(server, adminAuth, http.MethodGet, "/api/system/operators/"+operator.ID+"/sessions", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("list operator sessions status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var sessions []data.OperatorSession
	if err := json.NewDecoder(recorder.Body).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("operator session count = %d sessions = %#v", len(sessions), sessions)
	}
	for _, session := range sessions {
		if session.ID == "" || session.OperatorID != operator.ID || session.TokenHash != "" || session.Current {
			t.Fatalf("unexpected listed operator session: %#v", session)
		}
	}
}

func TestAdminRevokesSingleOperatorSession(t *testing.T) {
	repository, server, adminAuth := newAuthenticatedTestServer(t)
	operator := createOperatorForSessionTests(t, server, adminAuth)
	firstAuth := loginOperator(t, server, operator.Username, "secret123A")
	secondAuth := loginOperator(t, server, operator.Username, "secret123A")
	firstSessionID := repository.sessions[sessionTokenHash(firstAuth.session.Value)].ID

	missingCSRFRecorder := serveAuthenticatedWithoutCSRF(
		server,
		adminAuth,
		http.MethodDelete,
		"/api/system/operators/"+operator.ID+"/sessions/"+firstSessionID,
		"",
	)
	if missingCSRFRecorder.Code != http.StatusForbidden {
		t.Fatalf("missing csrf single session revoke status = %d body = %s", missingCSRFRecorder.Code, missingCSRFRecorder.Body.String())
	}

	recorder := serveAuthenticated(server, adminAuth, http.MethodDelete, "/api/system/operators/"+operator.ID+"/sessions/"+firstSessionID, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("single session revoke status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	revokedRecorder := serveAuthenticated(server, firstAuth, http.MethodGet, "/api/auth/me", "")
	if revokedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status = %d body = %s", revokedRecorder.Code, revokedRecorder.Body.String())
	}
	remainingRecorder := serveAuthenticated(server, secondAuth, http.MethodGet, "/api/auth/me", "")
	if remainingRecorder.Code != http.StatusOK {
		t.Fatalf("remaining session status = %d body = %s", remainingRecorder.Code, remainingRecorder.Body.String())
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.session_revoke", "operator_session", firstSessionID)
	if event.Outcome != "success" ||
		event.Metadata["operatorId"] != operator.ID ||
		event.Metadata["username"] != operator.Username {
		t.Fatalf("unexpected single session revoke audit event: %#v", event)
	}
}

func TestOperatorCannotRevokeCurrentSessionThroughAdminRoute(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	currentSessionID := repository.sessions[sessionTokenHash(auth.session.Value)].ID

	recorder := serveAuthenticated(server, auth, http.MethodDelete, "/api/system/operators/op_admin/sessions/"+currentSessionID, "")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("revoke current session through admin route status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "auth_current_session_revoke_forbidden" || response.Message != "current session cannot be revoked" {
		t.Fatalf("unexpected current session revoke response: %#v", response)
	}
	if _, err := repository.GetOperatorBySession(t.Context(), sessionTokenHash(auth.session.Value), time.Now().UTC()); err != nil {
		t.Fatalf("current operator session was revoked: %v", err)
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.session_revoke", "operator_session", currentSessionID)
	if event.Outcome != "failure" || event.Metadata["reason"] != "auth_current_session_revoke_forbidden" {
		t.Fatalf("unexpected current session revoke audit event: %#v", event)
	}
}

func createOperatorForSessionTests(t *testing.T, server http.Handler, auth *authTestSession) data.Operator {
	t.Helper()

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators",
		`{"username":"ops-session","password":"secret123A","enabled":true}`,
	)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create operator status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var operator data.Operator
	if err := json.NewDecoder(recorder.Body).Decode(&operator); err != nil {
		t.Fatal(err)
	}
	return operator
}
