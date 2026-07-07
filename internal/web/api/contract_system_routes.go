package api

import "net/http"

func addSystemContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/api/system/health", http.MethodGet, operation(
		"system", "systemHealth", "Read system health", http.StatusOK, schemaRef("SystemHealth"),
	))
	addOperation(paths, "/api/system/api-contract", http.MethodGet, operation(
		"system", "apiContract", "Read the OpenAPI schema contract", http.StatusOK,
		map[string]any{"type": "object", "description": "OpenAPI 3.1 contract document"},
	))
	addOperation(paths, "/api/system/notifications", http.MethodGet, operation(
		"system", "listNotifications", "List notifications", http.StatusOK, arraySchema(schemaRef("Notification")),
	))
	addOperation(paths, "/api/system/notifications/{id}/retry", http.MethodPost, operation(
		"system", "retryNotification", "Retry a failed notification", http.StatusOK, schemaRef("Notification"),
		withCSRF(), withParameters(pathParam("id", "Notification id")), withErrors(http.StatusNotFound, http.StatusConflict),
	))
	addNotificationChannelContractPaths(paths)
	addOperation(paths, "/api/system/exchange-accounts", http.MethodGet, operation(
		"system", "listExchangeAccounts", "List exchange accounts without secret material", http.StatusOK, arraySchema(schemaRef("ExchangeAccount")),
	))
	addOperation(paths, "/api/system/exchange-accounts", http.MethodPost, operation(
		"system", "createExchangeAccount", "Create an exchange account", http.StatusCreated, schemaRef("ExchangeAccount"),
		withCSRF(), withRequest(schemaRef("CreateExchangeAccount")), withErrors(http.StatusBadRequest),
	))
	addOperation(paths, "/api/system/operators", http.MethodGet, operation(
		"system", "listOperators", "List operators", http.StatusOK, arraySchema(schemaRef("Operator")),
	))
	addOperation(paths, "/api/system/operators", http.MethodPost, operation(
		"system", "createOperator", "Create an operator", http.StatusCreated, schemaRef("Operator"),
		withCSRF(), withRequest(schemaRef("CreateOperator")), withErrors(http.StatusBadRequest),
	))
	addSystemActionContractPaths(paths, "/api/system/operators/{id}",
		"setOperator", "Set operator", "Operator", "Operator id", "enable", "disable")
	addOperation(paths, "/api/system/audit-events", http.MethodGet, operation(
		"system", "listAuditEvents", "List operation audit events", http.StatusOK, arraySchema(schemaRef("AuditEvent")),
		withParameters(auditEventLimitQueryParam()),
	))
	addOperation(paths, "/api/system/audit-events/export", http.MethodGet, operation(
		"system", "exportAuditEvents", "Export operation audit events as CSV", http.StatusOK, map[string]any{"type": "string"},
		withResponseContentType("text/csv"),
		withParameters(auditEventLimitQueryParam()),
	))
}

func auditEventLimitQueryParam() apiParameter {
	return queryParam("limit", false, "Maximum number of events", map[string]any{"type": "integer", "minimum": 1, "maximum": 500})
}
