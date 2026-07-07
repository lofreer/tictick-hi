package api

import (
	"errors"
	"net/http"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (server *Server) handleOperators(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		operators, err := server.repository.ListOperators(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, operators)
	case http.MethodPost:
		actor, _, authErr := server.currentOperator(r)
		if authErr != nil {
			writeAuthError(w, authErr)
			return
		}
		if !actor.IsAdmin() {
			server.writeAdminRequiredAudit(w, r, actor, "operator.create", "operator", "", nil)
			return
		}
		var request data.CreateOperator
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validateOperator(request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		operator, err := server.repository.CreateOperator(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := server.recordAuditEvent(r, actor, "operator.create", "operator", operator.ID, "success", map[string]string{
			"username": operator.Username,
			"role":     operator.Role,
			"enabled":  boolString(operator.Enabled),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, operator)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (server *Server) handleOperatorAction(w http.ResponseWriter, r *http.Request, id string, action string) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	actor, _, authErr := server.currentOperator(r)
	if authErr != nil {
		writeAuthError(w, authErr)
		return
	}
	var enabled bool
	switch action {
	case "enable":
		enabled = true
	case "disable":
		enabled = false
	case "role":
		server.handleOperatorRoleAction(w, r, actor, id)
		return
	default:
		writeError(w, http.StatusNotFound, "operator action not found")
		return
	}
	if !actor.IsAdmin() {
		server.writeAdminRequiredAudit(w, r, actor, "operator."+action, "operator", id, nil)
		return
	}
	if !enabled && id == actor.ID {
		if err := server.recordAuditEvent(r, actor, "operator.disable", "operator", actor.ID, "failure", map[string]string{
			"username": actor.Username,
			"role":     actor.Role,
			"enabled":  boolString(actor.Enabled),
			"reason":   "self_disable_forbidden",
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeAPIError(w, http.StatusConflict, apiErrorOperatorSelfDisableForbidden, "current operator cannot be disabled")
		return
	}
	operator, err := server.repository.SetOperatorEnabled(r.Context(), id, enabled)
	if err != nil {
		if auditErr := server.recordAuditEvent(r, actor, "operator."+action, "operator", id, "failure", operatorStoreFailureMetadata(err, map[string]string{
			"enabled": boolString(enabled),
		})); auditErr != nil {
			writeError(w, http.StatusInternalServerError, auditErr.Error())
			return
		}
		writeStoreError(w, err)
		return
	}
	if err := server.recordAuditEvent(r, actor, "operator."+action, "operator", operator.ID, "success", map[string]string{
		"username": operator.Username,
		"role":     operator.Role,
		"enabled":  boolString(operator.Enabled),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, operator)
}

func (server *Server) handleOperatorRoleAction(
	w http.ResponseWriter,
	r *http.Request,
	actor data.Operator,
	id string,
) {
	if !actor.IsAdmin() {
		server.writeAdminRequiredAudit(w, r, actor, "operator.role", "operator", id, nil)
		return
	}
	var request data.UpdateOperatorRole
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	role := data.NormalizeOperatorRole(request.Role)
	if err := validateOperatorRoleUpdate(data.UpdateOperatorRole{Role: role}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if id == actor.ID && role != actor.Role {
		if err := server.recordAuditEvent(r, actor, "operator.role", "operator", actor.ID, "failure", map[string]string{
			"username":      actor.Username,
			"role":          actor.Role,
			"requestedRole": role,
			"reason":        "self_role_change_forbidden",
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeAPIError(w, http.StatusConflict, apiErrorOperatorSelfRoleChangeForbidden, "current operator role cannot be changed")
		return
	}
	result, err := server.repository.SetOperatorRole(r.Context(), id, role)
	if err != nil {
		if auditErr := server.recordAuditEvent(r, actor, "operator.role", "operator", id, "failure", operatorStoreFailureMetadata(err, map[string]string{
			"requestedRole": role,
		})); auditErr != nil {
			writeError(w, http.StatusInternalServerError, auditErr.Error())
			return
		}
		writeStoreError(w, err)
		return
	}
	if err := server.recordAuditEvent(r, actor, "operator.role", "operator", result.Operator.ID, "success", map[string]string{
		"username":     result.Operator.Username,
		"previousRole": result.PreviousRole,
		"role":         result.Operator.Role,
		"enabled":      boolString(result.Operator.Enabled),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result.Operator)
}

func operatorStoreFailureMetadata(err error, metadata map[string]string) map[string]string {
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["reason"] = operatorStoreFailureReason(err)
	return metadata
}

func operatorStoreFailureReason(err error) string {
	if code, ok := data.DomainErrorCode(err); ok {
		return string(code)
	}
	if errors.Is(err, data.ErrNotFound) {
		return "not_found"
	}
	if errors.Is(err, data.ErrInvalidState) {
		return "invalid_state"
	}
	return "store_error"
}

func (server *Server) writeAdminRequiredAudit(
	w http.ResponseWriter,
	r *http.Request,
	actor data.Operator,
	action string,
	resourceType string,
	resourceID string,
	metadata map[string]string,
) {
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["reason"] = "admin_required"
	metadata["actorRole"] = actor.Role
	if err := server.recordAuditEvent(r, actor, action, resourceType, resourceID, "failure", metadata); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeAPIError(w, http.StatusForbidden, apiErrorForbidden, "admin operator role is required")
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
