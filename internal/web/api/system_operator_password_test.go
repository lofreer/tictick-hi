package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestAdminResetsOperatorPasswordAndRevokesSessions(t *testing.T) {
	repository, server, adminAuth := newAuthenticatedTestServer(t)
	operator := createOperatorForSessionTests(t, server, adminAuth)
	firstAuth := loginOperator(t, server, operator.Username, "secret123A")
	secondAuth := loginOperator(t, server, operator.Username, "secret123A")

	recorder := serveAuthenticated(
		server,
		adminAuth,
		http.MethodPost,
		"/api/system/operators/"+operator.ID+"/password",
		`{"newPassword":"reset456B"}`,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("reset operator password status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.ResetOperatorPasswordResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.RevokedSessionCount != 2 {
		t.Fatalf("revoked session count = %d, want 2", result.RevokedSessionCount)
	}
	if oldLogin := loginOperatorRecorder(server, operator.Username, "secret123A"); oldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("old password login status = %d body = %s", oldLogin.Code, oldLogin.Body.String())
	}
	loginOperator(t, server, operator.Username, "reset456B")
	if revokedRecorder := serveAuthenticated(server, firstAuth, http.MethodGet, "/api/auth/me", ""); revokedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("first revoked session status = %d body = %s", revokedRecorder.Code, revokedRecorder.Body.String())
	}
	if revokedRecorder := serveAuthenticated(server, secondAuth, http.MethodGet, "/api/auth/me", ""); revokedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("second revoked session status = %d body = %s", revokedRecorder.Code, revokedRecorder.Body.String())
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.password_reset", "operator", operator.ID)
	if event.Outcome != "success" ||
		event.Metadata["revokedSessionCount"] != "2" ||
		event.Metadata["username"] != operator.Username {
		t.Fatalf("unexpected password reset audit event: %#v", event)
	}
}

func TestOperatorCannotResetOwnPasswordThroughAdminRoute(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/operators/op_admin/password", `{"newPassword":"reset456B"}`)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("reset own password status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "operator_self_password_reset_forbidden" ||
		response.Message != "current operator password cannot be reset here" {
		t.Fatalf("unexpected self password reset response: %#v", response)
	}
	if _, err := repository.GetOperatorBySession(t.Context(), sessionTokenHash(auth.session.Value), time.Now().UTC()); err != nil {
		t.Fatalf("current operator session was revoked: %v", err)
	}
	if newLogin := loginOperatorRecorder(server, testUsername, "reset456B"); newLogin.Code != http.StatusUnauthorized {
		t.Fatalf("self reset changed password, login status = %d body = %s", newLogin.Code, newLogin.Body.String())
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.password_reset", "operator", "op_admin")
	if event.Outcome != "failure" || event.Metadata["reason"] != "self_password_reset_forbidden" {
		t.Fatalf("unexpected self password reset audit event: %#v", event)
	}
}

func TestAdminResetOperatorPasswordRejectsPolicyAndHistoryFailures(t *testing.T) {
	repository, server, adminAuth := newAuthenticatedTestServer(t)
	operator := createOperatorForSessionTests(t, server, adminAuth)

	weakRecorder := serveAuthenticated(
		server,
		adminAuth,
		http.MethodPost,
		"/api/system/operators/"+operator.ID+"/password",
		`{"newPassword":"password123"}`,
	)
	if weakRecorder.Code != http.StatusBadRequest {
		t.Fatalf("weak password reset status = %d body = %s", weakRecorder.Code, weakRecorder.Body.String())
	}
	weakEvent := assertAuditAction(t, repository.auditEvents, "operator.password_reset", "operator", operator.ID)
	if weakEvent.Outcome != "failure" || weakEvent.Metadata["reason"] != "password_policy" {
		t.Fatalf("unexpected weak password reset audit event: %#v", weakEvent)
	}

	reuseRecorder := serveAuthenticated(
		server,
		adminAuth,
		http.MethodPost,
		"/api/system/operators/"+operator.ID+"/password",
		`{"newPassword":"secret123A"}`,
	)
	if reuseRecorder.Code != http.StatusBadRequest {
		t.Fatalf("reused password reset status = %d body = %s", reuseRecorder.Code, reuseRecorder.Body.String())
	}
	reuseResponse := decodeAPIError(t, reuseRecorder)
	if reuseResponse.Code != "operator_password_reused" {
		t.Fatalf("unexpected reused password response: %#v", reuseResponse)
	}
	reuseEvent := assertAuditAction(t, repository.auditEvents[len(repository.auditEvents)-1:], "operator.password_reset", "operator", operator.ID)
	if reuseEvent.Outcome != "failure" || reuseEvent.Metadata["reason"] != "password_history" {
		t.Fatalf("unexpected reused password reset audit event: %#v", reuseEvent)
	}
}

func TestOperatorPasswordResetRequiresAdminRole(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.operators = append(repository.operators, data.Operator{
		ID:        "op_ops",
		Username:  "ops",
		Role:      data.OperatorRoleOperator,
		Enabled:   true,
		CreatedAt: repository.operators[0].CreatedAt,
		UpdatedAt: repository.operators[0].UpdatedAt,
	})
	repository.passwords["op_ops"] = "secret123A"
	repository.operators[0].Role = data.OperatorRoleOperator

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/operators/op_ops/password", `{"newPassword":"reset456B"}`)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("non-admin password reset status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.password_reset", "operator", "op_ops")
	if event.Outcome != "failure" || event.Metadata["reason"] != "admin_required" ||
		event.Metadata["actorRole"] != data.OperatorRoleOperator {
		t.Fatalf("unexpected password reset admin required audit event: %#v", event)
	}
}

func loginOperatorRecorder(server http.Handler, username string, password string) *httptest.ResponseRecorder {
	body := bytes.NewBufferString(`{"username":"` + username + `","password":"` + password + `"}`)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", body))
	return recorder
}
