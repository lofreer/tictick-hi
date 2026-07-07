package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestOperatorCannotDisableSelf(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/operators/op_admin/disable", "")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("disable self status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "operator_self_disable_forbidden" || response.Message != "current operator cannot be disabled" {
		t.Fatalf("unexpected self disable response: %#v", response)
	}

	operator, err := repository.AuthenticateOperator(t.Context(), testUsername, testPassword)
	if err != nil {
		t.Fatalf("current operator was disabled: %v", err)
	}
	if !operator.Enabled {
		t.Fatalf("current operator enabled = false")
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.disable", "operator", "op_admin")
	if event.Outcome != "failure" {
		t.Fatalf("self disable audit outcome = %q, want failure: %#v", event.Outcome, event)
	}
	if event.Metadata["reason"] != "self_disable_forbidden" || event.Metadata["username"] != testUsername {
		t.Fatalf("unexpected self disable audit metadata: %#v", event.Metadata)
	}
}

func TestOperatorManagementRequiresAdminRole(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.operators[0].Role = data.OperatorRoleOperator

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators",
		`{"username":"ops","password":"secret123A","enabled":true}`,
	)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("create operator status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "forbidden" || response.Message != "admin operator role is required" {
		t.Fatalf("unexpected admin required response: %#v", response)
	}
	if len(repository.operators) != 1 {
		t.Fatalf("operator was created by non-admin: %#v", repository.operators)
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.create", "operator", "")
	if event.Outcome != "failure" || event.Metadata["reason"] != "admin_required" || event.Metadata["actorRole"] != data.OperatorRoleOperator {
		t.Fatalf("unexpected admin required audit event: %#v", event)
	}
}

func TestCreateOperatorRejectsWeakPassword(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	tests := []struct {
		name     string
		password string
		message  string
	}{
		{
			name:     "too short",
			password: "short",
			message:  "password must be at least 8 characters",
		},
		{
			name:     "missing digit",
			password: "password",
			message:  "password must include at least one letter and one number",
		},
		{
			name:     "common",
			password: "secret123",
			message:  "password is too common",
		},
		{
			name:     "contains username",
			password: "weakPass123",
			message:  "password must not include the username",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := serveAuthenticated(
				server,
				auth,
				http.MethodPost,
				"/api/system/operators",
				`{"username":"weak","password":"`+test.password+`","enabled":true}`,
			)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("weak password status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			response := decodeAPIError(t, recorder)
			if response.Message != test.message {
				t.Fatalf("unexpected weak password response: %#v", response)
			}
		})
	}
}

func TestCreateOperatorRejectsInvalidRole(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators",
		`{"username":"ops","password":"secret123A","role":"owner","enabled":true}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid role status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Message != "operator role must be admin or operator" {
		t.Fatalf("unexpected invalid role response: %#v", response)
	}
}

func TestCreateOperatorRejectsBlankUsername(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators",
		`{"username":"   ","password":"secret123A","enabled":true}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("blank username status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Message != "username and password are required" {
		t.Fatalf("unexpected blank username response: %#v", response)
	}
}

func TestRepositoryRejectsDisablingLastEnabledOperator(t *testing.T) {
	repository := newFakeRepository()

	_, err := repository.SetOperatorEnabled(t.Context(), "op_admin", false)
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("SetOperatorEnabled error = %v, want invalid state", err)
	}
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeOperatorLastEnabledRequired {
		t.Fatalf("SetOperatorEnabled code = %q, %t; want %q, true", code, ok, data.ErrorCodeOperatorLastEnabledRequired)
	}
	if _, err := repository.AuthenticateOperator(t.Context(), testUsername, testPassword); err != nil {
		t.Fatalf("last enabled operator was disabled: %v", err)
	}
}

func TestRepositoryRejectsDisablingLastEnabledAdmin(t *testing.T) {
	repository := newFakeRepository()
	repository.operators = append(repository.operators, data.Operator{
		ID:        "op_ops",
		Username:  "ops",
		Role:      data.OperatorRoleOperator,
		Enabled:   true,
		CreatedAt: repository.operators[0].CreatedAt,
		UpdatedAt: repository.operators[0].UpdatedAt,
	})

	_, err := repository.SetOperatorEnabled(t.Context(), "op_admin", false)
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("SetOperatorEnabled error = %v, want invalid state", err)
	}
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeOperatorLastAdminRequired {
		t.Fatalf("SetOperatorEnabled code = %q, %t; want %q, true", code, ok, data.ErrorCodeOperatorLastAdminRequired)
	}
}
