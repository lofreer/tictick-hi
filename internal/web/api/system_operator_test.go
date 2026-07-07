package api

import (
	"context"
	"encoding/json"
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

	listRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/operators", "")
	if listRecorder.Code != http.StatusForbidden {
		t.Fatalf("list operators status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	listResponse := decodeAPIError(t, listRecorder)
	if listResponse.Code != "forbidden" || listResponse.Message != "admin operator role is required" {
		t.Fatalf("unexpected list admin required response: %#v", listResponse)
	}
	listEvent := assertAuditAction(t, repository.auditEvents, "operator.list", "operator", "")
	if listEvent.Outcome != "failure" || listEvent.Metadata["reason"] != "admin_required" ||
		listEvent.Metadata["actorRole"] != data.OperatorRoleOperator {
		t.Fatalf("unexpected list admin required audit event: %#v", listEvent)
	}

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

	roleRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators/op_admin/role",
		`{"role":"admin"}`,
	)
	if roleRecorder.Code != http.StatusForbidden {
		t.Fatalf("role update status = %d body = %s", roleRecorder.Code, roleRecorder.Body.String())
	}
	roleResponse := decodeAPIError(t, roleRecorder)
	if roleResponse.Code != "forbidden" || roleResponse.Message != "admin operator role is required" {
		t.Fatalf("unexpected role admin required response: %#v", roleResponse)
	}
	roleEvent := assertAuditAction(t, repository.auditEvents, "operator.role", "operator", "op_admin")
	if roleEvent.Outcome != "failure" || roleEvent.Metadata["reason"] != "admin_required" ||
		roleEvent.Metadata["actorRole"] != data.OperatorRoleOperator {
		t.Fatalf("unexpected role admin required audit event: %#v", roleEvent)
	}
}

func TestDisablingOperatorRevokesSessions(t *testing.T) {
	repository, server, adminAuth := newAuthenticatedTestServer(t)
	createRecorder := serveAuthenticated(
		server,
		adminAuth,
		http.MethodPost,
		"/api/system/operators",
		`{"username":"ops-revoke","password":"secret123A","enabled":true}`,
	)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create operator status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}
	var operator data.Operator
	if err := json.NewDecoder(createRecorder.Body).Decode(&operator); err != nil {
		t.Fatal(err)
	}

	targetAuth := loginOperator(t, server, "ops-revoke", "secret123A")
	targetTokenHash := sessionTokenHash(targetAuth.session.Value)
	if _, exists := repository.sessions[targetTokenHash]; !exists {
		t.Fatalf("target session was not created: %#v", repository.sessions)
	}

	disableRecorder := serveAuthenticated(server, adminAuth, http.MethodPost, "/api/system/operators/"+operator.ID+"/disable", "")
	if disableRecorder.Code != http.StatusOK {
		t.Fatalf("disable operator status = %d body = %s", disableRecorder.Code, disableRecorder.Body.String())
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.disable", "operator", operator.ID)
	if event.Outcome != "success" || event.Metadata["revokedSessionCount"] != "1" {
		t.Fatalf("unexpected disable audit event: %#v", event)
	}
	if _, exists := repository.sessions[targetTokenHash]; exists {
		t.Fatalf("disabled operator session was not revoked: %#v", repository.sessions[targetTokenHash])
	}
	meRecorder := serveAuthenticated(server, targetAuth, http.MethodGet, "/api/auth/me", "")
	if meRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("disabled operator session status = %d body = %s", meRecorder.Code, meRecorder.Body.String())
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

func TestOperatorRoleUpdateAuditsPreviousRole(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.operators = append(repository.operators, data.Operator{
		ID:        "op_ops",
		Username:  "ops",
		Role:      data.OperatorRoleOperator,
		Enabled:   true,
		CreatedAt: repository.operators[0].CreatedAt,
		UpdatedAt: repository.operators[0].UpdatedAt,
	})

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators/op_ops/role",
		`{"role":"admin"}`,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("role update status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var updated data.Operator
	if err := json.NewDecoder(recorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != "op_ops" || updated.Role != data.OperatorRoleAdmin {
		t.Fatalf("unexpected updated operator: %#v", updated)
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.role", "operator", "op_ops")
	if event.Outcome != "success" ||
		event.Metadata["previousRole"] != data.OperatorRoleOperator ||
		event.Metadata["role"] != data.OperatorRoleAdmin ||
		event.Metadata["username"] != "ops" {
		t.Fatalf("unexpected role audit metadata: %#v", event)
	}
}

func TestOperatorCannotChangeOwnRole(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators/op_admin/role",
		`{"role":"operator"}`,
	)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("self role update status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "operator_self_role_change_forbidden" ||
		response.Message != "current operator role cannot be changed" {
		t.Fatalf("unexpected self role response: %#v", response)
	}
	if repository.operators[0].Role != data.OperatorRoleAdmin {
		t.Fatalf("current operator role changed: %#v", repository.operators[0])
	}
	event := assertAuditAction(t, repository.auditEvents, "operator.role", "operator", "op_admin")
	if event.Outcome != "failure" ||
		event.Metadata["reason"] != "self_role_change_forbidden" ||
		event.Metadata["requestedRole"] != data.OperatorRoleOperator {
		t.Fatalf("unexpected self role audit metadata: %#v", event)
	}
}

func TestUpdateOperatorRoleRejectsInvalidRole(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators/op_admin/role",
		`{"role":"owner"}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid role update status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Message != "operator role must be admin or operator" {
		t.Fatalf("unexpected invalid role update response: %#v", response)
	}
}

func TestOperatorEnableDisableStoreFailureAudited(t *testing.T) {
	base := newFakeRepository()
	repository := &operatorActionFailureRepository{
		fakeRepository: base,
		enabledErr:     data.OperatorLastAdminError(),
	}
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/operators/op_target/disable", "")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("disable status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "operator_last_admin_required" {
		t.Fatalf("unexpected disable response: %#v", response)
	}
	event := assertAuditAction(t, base.auditEvents, "operator.disable", "operator", "op_target")
	if event.Outcome != "failure" ||
		event.Metadata["reason"] != string(data.ErrorCodeOperatorLastAdminRequired) ||
		event.Metadata["enabled"] != "false" {
		t.Fatalf("unexpected disable failure audit metadata: %#v", event)
	}
}

func TestOperatorRoleStoreFailureAudited(t *testing.T) {
	base := newFakeRepository()
	repository := &operatorActionFailureRepository{
		fakeRepository: base,
		roleErr:        data.OperatorLastAdminError(),
	}
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators/op_target/role",
		`{"role":"operator"}`,
	)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("role status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "operator_last_admin_required" {
		t.Fatalf("unexpected role response: %#v", response)
	}
	event := assertAuditAction(t, base.auditEvents, "operator.role", "operator", "op_target")
	if event.Outcome != "failure" ||
		event.Metadata["reason"] != string(data.ErrorCodeOperatorLastAdminRequired) ||
		event.Metadata["requestedRole"] != data.OperatorRoleOperator {
		t.Fatalf("unexpected role failure audit metadata: %#v", event)
	}
}

func TestCreateOperatorStoreFailureAudited(t *testing.T) {
	base := newFakeRepository()
	repository := &operatorActionFailureRepository{
		fakeRepository: base,
		createErr:      errors.New("stage8 create operator store failure"),
	}
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators",
		`{"username":"ops-store","password":"secret123A","enabled":true}`,
	)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("create status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(base.operators) != 1 {
		t.Fatalf("operator was created despite store failure: %#v", base.operators)
	}
	event := assertAuditAction(t, base.auditEvents, "operator.create", "operator", "")
	if event.Outcome != "failure" ||
		event.Metadata["reason"] != "store_error" ||
		event.Metadata["username"] != "ops-store" ||
		event.Metadata["role"] != data.OperatorRoleOperator ||
		event.Metadata["enabled"] != "true" {
		t.Fatalf("unexpected create failure audit metadata: %#v", event.Metadata)
	}
	if _, ok := event.Metadata["password"]; ok {
		t.Fatalf("operator create failure audit metadata includes password: %#v", event.Metadata)
	}
	for key, value := range event.Metadata {
		if value == "secret123A" {
			t.Fatalf("operator create failure audit metadata leaked password in %s=%q", key, value)
		}
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

func TestRepositoryRejectsDemotingLastEnabledAdmin(t *testing.T) {
	repository := newFakeRepository()
	repository.operators = append(repository.operators, data.Operator{
		ID:        "op_ops",
		Username:  "ops",
		Role:      data.OperatorRoleOperator,
		Enabled:   true,
		CreatedAt: repository.operators[0].CreatedAt,
		UpdatedAt: repository.operators[0].UpdatedAt,
	})

	_, err := repository.SetOperatorRole(t.Context(), "op_admin", data.OperatorRoleOperator)
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("SetOperatorRole error = %v, want invalid state", err)
	}
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeOperatorLastAdminRequired {
		t.Fatalf("SetOperatorRole code = %q, %t; want %q, true", code, ok, data.ErrorCodeOperatorLastAdminRequired)
	}
	if repository.operators[0].Role != data.OperatorRoleAdmin {
		t.Fatalf("last admin role changed: %#v", repository.operators[0])
	}
}

type operatorActionFailureRepository struct {
	*fakeRepository
	createErr  error
	enabledErr error
	roleErr    error
}

func (repository *operatorActionFailureRepository) CreateOperator(
	ctx context.Context,
	request data.CreateOperator,
) (data.Operator, error) {
	if repository.createErr == nil {
		return repository.fakeRepository.CreateOperator(ctx, request)
	}
	return data.Operator{}, repository.createErr
}

func (repository *operatorActionFailureRepository) SetOperatorEnabled(
	_ context.Context,
	_ string,
	_ bool,
) (data.Operator, error) {
	return data.Operator{}, repository.enabledErr
}

func (repository *operatorActionFailureRepository) SetOperatorRole(
	_ context.Context,
	_ string,
	_ string,
) (data.OperatorRoleUpdateResult, error) {
	return data.OperatorRoleUpdateResult{}, repository.roleErr
}
