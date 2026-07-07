package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestChangeOperatorPasswordUpdatesPasswordAndRevokesOtherSessions(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")
	currentAuth := loginTestOperator(t, server)
	otherAuth := loginTestOperator(t, server)

	recorder := serveAuthenticated(
		server,
		currentAuth,
		http.MethodPost,
		"/api/auth/password",
		`{"currentPassword":"`+testPassword+`","newPassword":"secret456B"}`,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("password change status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.ChangeOperatorPasswordResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" || result.RevokedSessionCount != 1 {
		t.Fatalf("unexpected password change result: %#v", result)
	}
	if _, err := repository.AuthenticateOperator(t.Context(), testUsername, testPassword); err != data.ErrUnauthorized {
		t.Fatalf("old password authentication error = %v, want unauthorized", err)
	}
	if _, err := repository.AuthenticateOperator(t.Context(), testUsername, "secret456B"); err != nil {
		t.Fatalf("new password authentication failed: %v", err)
	}
	currentRecorder := serveAuthenticated(server, currentAuth, http.MethodGet, "/api/auth/me", "")
	if currentRecorder.Code != http.StatusOK {
		t.Fatalf("current session status = %d body = %s", currentRecorder.Code, currentRecorder.Body.String())
	}
	otherRecorder := serveAuthenticated(server, otherAuth, http.MethodGet, "/api/auth/me", "")
	if otherRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("other session status = %d body = %s", otherRecorder.Code, otherRecorder.Body.String())
	}
	event := assertAuditAction(t, repository.auditEvents, "auth.password_change", "operator", "op_admin")
	if event.Outcome != "success" || event.Metadata["revokedSessionCount"] != "1" {
		t.Fatalf("unexpected password change audit event: %#v", event)
	}
}

func TestChangeOperatorPasswordRejectsWeakNewPassword(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/auth/password",
		`{"currentPassword":"`+testPassword+`","newPassword":"short1"}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("weak password status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if _, err := repository.AuthenticateOperator(t.Context(), testUsername, testPassword); err != nil {
		t.Fatalf("current password should remain valid: %v", err)
	}
	event := assertAuditAction(t, repository.auditEvents, "auth.password_change", "operator", "op_admin")
	if event.Outcome != "failure" || event.Metadata["reason"] != "password_policy" {
		t.Fatalf("unexpected weak password audit event: %#v", event)
	}
}

func TestChangeOperatorPasswordRejectsInvalidCurrentPassword(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/auth/password",
		`{"currentPassword":"wrong123A","newPassword":"secret456B"}`,
	)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid current password status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if _, err := repository.AuthenticateOperator(t.Context(), testUsername, testPassword); err != nil {
		t.Fatalf("current password should remain valid: %v", err)
	}
	event := assertAuditAction(t, repository.auditEvents, "auth.password_change", "operator", "op_admin")
	if event.Outcome != "failure" || event.Metadata["reason"] != "invalid_current_password" {
		t.Fatalf("unexpected invalid current password audit event: %#v", event)
	}
}

func TestChangeOperatorPasswordRejectsRecentPasswordReuse(t *testing.T) {
	repository := newFakeRepository()
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	firstRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/auth/password",
		`{"currentPassword":"`+testPassword+`","newPassword":"secret456B"}`,
	)
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("first password change status = %d body = %s", firstRecorder.Code, firstRecorder.Body.String())
	}
	reuseRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/auth/password",
		`{"currentPassword":"secret456B","newPassword":"`+testPassword+`"}`,
	)
	if reuseRecorder.Code != http.StatusBadRequest {
		t.Fatalf("reused password status = %d body = %s", reuseRecorder.Code, reuseRecorder.Body.String())
	}
	response := decodeAPIError(t, reuseRecorder)
	if response.Code != "operator_password_reused" ||
		response.Message != "new password must not reuse a recent operator password" {
		t.Fatalf("unexpected reused password response: %#v", response)
	}
	if _, err := repository.AuthenticateOperator(t.Context(), testUsername, "secret456B"); err != nil {
		t.Fatalf("current password should remain valid: %v", err)
	}
	event := repository.auditEvents[len(repository.auditEvents)-1]
	if event.Action != "auth.password_change" || event.Outcome != "failure" ||
		event.Metadata["reason"] != "password_history" {
		t.Fatalf("unexpected reused password audit event: %#v", event)
	}
}

func TestChangeOperatorPasswordRequiresCSRF(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticatedWithoutCSRF(
		server,
		auth,
		http.MethodPost,
		"/api/auth/password",
		`{"currentPassword":"`+testPassword+`","newPassword":"secret456B"}`,
	)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}
